package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

type authService interface {
	Register(ctx context.Context, email, password string) (TokenPair, error)
	Login(ctx context.Context, email, password string) (TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (TokenPair, error)
}

type Handler struct {
	svc authService
}

func NewHandler(svc authService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(public, _, _ *gin.RouterGroup) {
	g := public.Group("/auth")
	g.POST("/register", h.register)
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
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
