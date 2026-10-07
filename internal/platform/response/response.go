package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
)

type ErrDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Field   string         `json:"field,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type Envelope struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      any         `json:"data,omitempty"`
	Errors    []ErrDetail `json:"errors,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

func requestID(c *gin.Context) string {
	return reqcontext.RequestIDFromContext(c.Request.Context())
}

func Success(c *gin.Context, status int, msg string, data any) {
	c.JSON(status, Envelope{
		Success:   true,
		Message:   msg,
		Data:      data,
		RequestID: requestID(c),
	})
}

func Error(c *gin.Context, err error) {
	rid := requestID(c)

	var ae *apperr.AppError
	if errors.As(err, &ae) {
		c.JSON(ae.HTTPStatus, Envelope{
			Success:   false,
			Message:   ae.Message,
			Errors:    []ErrDetail{{Code: ae.Code, Message: ae.Message, Details: ae.Details}},
			RequestID: rid,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, Envelope{
		Success:   false,
		Message:   "internal server error",
		Errors:    []ErrDetail{{Code: apperr.CodeInternalError, Message: "internal server error"}},
		RequestID: rid,
	})
}

func Paginated(c *gin.Context, items any, meta pagination.Meta) {
	c.JSON(http.StatusOK, Envelope{
		Success: true,
		Message: "ok",
		Data: gin.H{
			"items":      items,
			"pagination": meta,
		},
		RequestID: requestID(c),
	})
}

// ErrorHandler renders any error pushed via c.Error() through the envelope.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 && !c.Writer.Written() {
			Error(c, c.Errors.Last().Err)
		}
	}
}
