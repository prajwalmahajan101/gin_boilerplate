package items

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

// itemService is the consumer-defined interface the handler depends on, so the
// handler is unit-testable with a stub. *ItemService satisfies it.
type itemService interface {
	Create(ctx context.Context, m *Item) error
	Update(ctx context.Context, m *Item) error
	Delete(ctx context.Context, id int64, soft bool) error
	GetByIDOrFail(ctx context.Context, id int64) (Item, error)
	List(ctx context.Context, page, pageSize int) ([]Item, pagination.Meta, error)
}

type Handler struct {
	svc itemService
}

func NewHandler(svc itemService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts item routes on the protected gate (auth attaches in M6).
func (h *Handler) RegisterRoutes(_, protected, _ *gin.RouterGroup) {
	g := protected.Group("/items")
	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.PATCH("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

type createReq struct {
	Name  string          `json:"name" binding:"required"`
	Code  string          `json:"code" binding:"required"`
	Notes json.RawMessage `json:"notes" swaggertype:"object"`
}

type updateReq struct {
	Name  *string         `json:"name"`
	Notes json.RawMessage `json:"notes" swaggertype:"object"`
}

type itemDTO struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Code      string          `json:"code"`
	Notes     json.RawMessage `json:"notes,omitempty" swaggertype:"object"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func toDTO(it Item) itemDTO {
	return itemDTO{
		ID:        it.ID,
		Name:      it.Name,
		Code:      it.Code,
		Notes:     it.Notes,
		IsActive:  it.IsActive,
		CreatedAt: it.CreatedAt,
		UpdatedAt: it.UpdatedAt,
	}
}

func parseID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

// create godoc
// @Summary   Create item
// @Tags      items
// @Accept    json
// @Produce   json
// @Param     body  body      createReq  true  "item to create"
// @Success   201   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Failure   409   {object}  response.Envelope
// @Router    /items [post]
func (h *Handler) create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	it := &Item{Name: req.Name, Code: req.Code, Notes: req.Notes}
	if err := h.svc.Create(c.Request.Context(), it); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "item created", toDTO(*it))
}

// list godoc
// @Summary   List items
// @Tags      items
// @Produce   json
// @Param     page       query     int  false  "page number"
// @Param     page_size  query     int  false  "page size"
// @Success   200        {object}  response.Envelope
// @Router    /items [get]
func (h *Handler) list(c *gin.Context) {
	page, size := pagination.Params(c)
	items, meta, err := h.svc.List(c.Request.Context(), page, size)
	if err != nil {
		response.Error(c, err)
		return
	}
	dtos := make([]itemDTO, len(items))
	for i, it := range items {
		dtos[i] = toDTO(it)
	}
	response.Paginated(c, dtos, meta)
}

// get godoc
// @Summary   Get item by ID
// @Tags      items
// @Produce   json
// @Param     id   path      int  true  "item id"
// @Success   200  {object}  response.Envelope
// @Failure   404  {object}  response.Envelope
// @Router    /items/{id} [get]
func (h *Handler) get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	it, err := h.svc.GetByIDOrFail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "ok", toDTO(it))
}

// update godoc
// @Summary   Update item
// @Tags      items
// @Accept    json
// @Produce   json
// @Param     id    path      int        true  "item id"
// @Param     body  body      updateReq  true  "fields to update"
// @Success   200   {object}  response.Envelope
// @Failure   400   {object}  response.Envelope
// @Failure   404   {object}  response.Envelope
// @Router    /items/{id} [patch]
func (h *Handler) update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	it, err := h.svc.GetByIDOrFail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.ValidationError(err.Error()))
		return
	}
	if req.Name != nil {
		it.Name = *req.Name
	}
	if req.Notes != nil {
		it.Notes = req.Notes
	}
	if err := h.svc.Update(c.Request.Context(), &it); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "item updated", toDTO(it))
}

// delete godoc
// @Summary   Soft-delete item
// @Tags      items
// @Produce   json
// @Param     id   path      int  true  "item id"
// @Success   200  {object}  response.Envelope
// @Failure   404  {object}  response.Envelope
// @Router    /items/{id} [delete]
func (h *Handler) delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, apperr.ValidationError("invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, true); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, "item deleted", nil)
}
