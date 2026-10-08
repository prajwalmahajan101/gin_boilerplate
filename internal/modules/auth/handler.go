package auth

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

type authService interface {
	Register(ctx context.Context, email, password string) (TokenPair, error)
	Login(ctx context.Context, email, password string) (TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (TokenPair, error)
	Logout(ctx context.Context, accessJTI, refreshJTI string) error
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error
	GetByIDOrFail(ctx context.Context, id int64) (User, error)
	List(ctx context.Context, page, pageSize int) ([]User, pagination.Meta, error)
	Update(ctx context.Context, m *User) error
	Delete(ctx context.Context, id int64, soft bool) error
}

type apiKeyService interface {
	Generate(ctx context.Context, userID int64, name string, expiresAt *time.Time) (APIKeyResult, error)
	ListByUser(ctx context.Context, userID int64) ([]APIKeyResult, error)
	Revoke(ctx context.Context, id int64) error
}

type Handler struct {
	svc    authService
	apiKey apiKeyService
	tokens *TokenService
}

func NewHandler(svc authService, apiKey apiKeyService, tokens *TokenService) *Handler {
	return &Handler{svc: svc, apiKey: apiKey, tokens: tokens}
}

func (h *Handler) RegisterRoutes(public, protected, admin *gin.RouterGroup) {
	g := public.Group("/auth")
	g.POST("/register", h.register)
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)

	// Authenticated auth actions
	protected.POST("/auth/logout", h.logout)
	protected.POST("/auth/change-password", h.changePassword)

	// Self-service API key management (authenticated users)
	k := protected.Group("/auth/api-keys")
	k.POST("", h.createOwnKey)
	k.GET("", h.listOwnKeys)
	k.DELETE("/:id", h.revokeOwnKey)

	// Admin user management
	u := admin.Group("/users")
	u.GET("", h.listUsers)
	u.GET("/:id", h.getUser)
	u.PATCH("/:id", h.updateUser)
	u.DELETE("/:id", h.deleteUser)

	// Admin API key management
	ak := admin.Group("/api-keys")
	ak.POST("", h.createKeyAdmin)
	ak.GET("", h.listKeysAdmin)
	ak.DELETE("/:id", h.revokeKeyAdmin)
}

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// register godoc
// @Summary   Register a new user
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     body  body      registerReq  true  "registration payload"
// @Success   201   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Failure   409   {object}  response.Envelope
// @Router    /auth/register [post]
func (h *Handler) register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	pair, err := h.svc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "registered", pair)
}

// login godoc
// @Summary   Log in with email and password
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     body  body      loginReq  true  "login payload"
// @Success   200   {object}  response.Envelope
// @Failure   401   {object}  response.Envelope
// @Router    /auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "logged in", pair)
}

// refresh godoc
// @Summary   Refresh token pair
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     body  body      refreshReq  true  "refresh payload"
// @Success   200   {object}  response.Envelope
// @Failure   401   {object}  response.Envelope
// @Router    /auth/refresh [post]
func (h *Handler) refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	pair, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "refreshed", pair)
}

// --- logout + password change ---

type logoutReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func extractBearerJTI(c *gin.Context, tokens *TokenService) string {
	hdr := c.GetHeader("Authorization")
	if len(hdr) > 7 {
		claims, err := tokens.ParseAccess(hdr[7:])
		if err == nil {
			return claims.JTI
		}
	}
	return ""
}

// logout godoc
// @Summary   Log out (revoke tokens)
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     body  body      logoutReq  true  "refresh token to revoke"
// @Success   200   {object}  response.Envelope
// @Failure   401   {object}  response.Envelope
// @Router    /auth/logout [post]
func (h *Handler) logout(c *gin.Context) {
	var req logoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	accessJTI := extractBearerJTI(c, h.tokens)
	rc, err := h.tokens.ParseRefresh(req.RefreshToken)
	if err != nil {
		response.Error(c, apperr.Unauthorized("invalid refresh token"))
		return
	}
	if err := h.svc.Logout(c.Request.Context(), accessJTI, rc.JTI); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "logged out", nil)
}

// changePassword godoc
// @Summary   Change password (authenticated)
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     body  body      changePasswordReq  true  "old and new passwords"
// @Success   200   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Failure   401   {object}  response.Envelope
// @Router    /auth/change-password [post]
func (h *Handler) changePassword(c *gin.Context) {
	uid, ok := callerID(c)
	if !ok {
		response.Error(c, apperr.Unauthorized("not authenticated"))
		return
	}
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "password changed", nil)
}

// --- admin user CRUD ---

type userDTO struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login_at,omitempty"`
}

func toUserDTO(u User) userDTO {
	return userDTO{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		LastLogin: u.LastLoginAt,
	}
}

type adminUpdateReq struct {
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

func parseID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

// listUsers godoc
// @Summary   List users (admin)
// @Tags      admin
// @Produce   json
// @Param     page       query     int  false  "page number"
// @Param     page_size  query     int  false  "page size"
// @Success   200        {object}  response.Envelope
// @Failure   401        {object}  response.Envelope
// @Failure   403        {object}  response.Envelope
// @Router    /admin/users [get]
func (h *Handler) listUsers(c *gin.Context) {
	page, size := pagination.Params(c)
	users, meta, err := h.svc.List(c.Request.Context(), page, size)
	if err != nil {
		response.Error(c, err)
		return
	}
	dtos := make([]userDTO, len(users))
	for i, u := range users {
		dtos[i] = toUserDTO(u)
	}
	response.Paginated(c, dtos, meta)
}

// getUser godoc
// @Summary   Get user by ID (admin)
// @Tags      admin
// @Produce   json
// @Param     id   path      int  true  "user id"
// @Success   200  {object}  response.Envelope
// @Failure   404  {object}  response.Envelope
// @Router    /admin/users/{id} [get]
func (h *Handler) getUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	u, err := h.svc.GetByIDOrFail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "ok", toUserDTO(u))
}

// updateUser godoc
// @Summary   Update user role or status (admin)
// @Tags      admin
// @Accept    json
// @Produce   json
// @Param     id    path      int             true  "user id"
// @Param     body  body      adminUpdateReq  true  "fields to update"
// @Success   200   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Failure   404   {object}  response.Envelope
// @Router    /admin/users/{id} [patch]
func (h *Handler) updateUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	u, err := h.svc.GetByIDOrFail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req adminUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	if req.Role != nil {
		if *req.Role != RoleUser && *req.Role != RoleAdmin {
			response.Error(c, apperr.ValidationError("role must be 'user' or 'admin'"))
			return
		}
		u.Role = *req.Role
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
	}
	if err := h.svc.Update(c.Request.Context(), &u); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "user updated", toUserDTO(u))
}

// deleteUser godoc
// @Summary   Soft-delete user (admin)
// @Tags      admin
// @Produce   json
// @Param     id   path      int  true  "user id"
// @Success   200  {object}  response.Envelope
// @Failure   404  {object}  response.Envelope
// @Router    /admin/users/{id} [delete]
func (h *Handler) deleteUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, true); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "user deleted", nil)
}

// --- API key endpoints ---

type createKeyReq struct {
	Name      string     `json:"name" binding:"required"`
	UserID    *int64     `json:"user_id"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func callerID(c *gin.Context) (int64, bool) {
	claims, ok := reqcontext.AuthFromContext(c.Request.Context())
	return claims.UserID, ok
}

// createOwnKey godoc
// @Summary   Create API key for self
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     body  body      createKeyReq  true  "key details"
// @Success   201   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Router    /auth/api-keys [post]
func (h *Handler) createOwnKey(c *gin.Context) {
	uid, ok := callerID(c)
	if !ok {
		response.Error(c, apperr.Unauthorized("not authenticated"))
		return
	}
	var req createKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	result, err := h.apiKey.Generate(c.Request.Context(), uid, req.Name, req.ExpiresAt)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "api key created", result)
}

// listOwnKeys godoc
// @Summary   List own API keys
// @Tags      auth
// @Produce   json
// @Success   200  {object}  response.Envelope
// @Router    /auth/api-keys [get]
func (h *Handler) listOwnKeys(c *gin.Context) {
	uid, ok := callerID(c)
	if !ok {
		response.Error(c, apperr.Unauthorized("not authenticated"))
		return
	}
	keys, err := h.apiKey.ListByUser(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "ok", keys)
}

// revokeOwnKey godoc
// @Summary   Revoke own API key
// @Tags      auth
// @Produce   json
// @Param     id   path      int  true  "api key id"
// @Success   200  {object}  response.Envelope
// @Router    /auth/api-keys/{id} [delete]
func (h *Handler) revokeOwnKey(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	if err := h.apiKey.Revoke(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "api key revoked", nil)
}

// createKeyAdmin godoc
// @Summary   Create API key for any user (admin)
// @Tags      admin
// @Accept    json
// @Produce   json
// @Param     body  body      createKeyReq  true  "key details (user_id required)"
// @Success   201   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Router    /admin/api-keys [post]
func (h *Handler) createKeyAdmin(c *gin.Context) {
	var req createKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	if req.UserID == nil {
		response.Error(c, apperr.ValidationError("user_id is required"))
		return
	}
	result, err := h.apiKey.Generate(c.Request.Context(), *req.UserID, req.Name, req.ExpiresAt)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "api key created", result)
}

// listKeysAdmin godoc
// @Summary   List API keys by user (admin)
// @Tags      admin
// @Produce   json
// @Param     user_id  query     int  true  "user id"
// @Success   200      {object}  response.Envelope
// @Router    /admin/api-keys [get]
func (h *Handler) listKeysAdmin(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil {
		response.Error(c, apperr.ValidationError("user_id query param required"))
		return
	}
	keys, err := h.apiKey.ListByUser(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "ok", keys)
}

// revokeKeyAdmin godoc
// @Summary   Revoke API key (admin)
// @Tags      admin
// @Produce   json
// @Param     id   path      int  true  "api key id"
// @Success   200  {object}  response.Envelope
// @Router    /admin/api-keys/{id} [delete]
func (h *Handler) revokeKeyAdmin(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	if err := h.apiKey.Revoke(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "api key revoked", nil)
}
