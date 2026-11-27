package httpapi

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/guilhermequitto/devices-crud/internal/device"
)

// errBadRequest marks request-level problems (malformed body, invalid id)
// that never reach the service.
var errBadRequest = errors.New("bad request")

// writeError maps an error to its HTTP status and writes an errorResponse.
// Unknown errors become 500 with a generic message; the detail is logged.
func writeError(c *gin.Context, err error) {
	status := statusFor(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		msg = "internal server error"
	}
	c.AbortWithStatusJSON(status, errorResponse{Error: msg})
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, errBadRequest),
		errors.Is(err, device.ErrInvalidName),
		errors.Is(err, device.ErrInvalidBrand),
		errors.Is(err, device.ErrInvalidState),
		errors.Is(err, device.ErrInvalidPatch):
		return http.StatusBadRequest
	case errors.Is(err, device.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, device.ErrInUseUpdate),
		errors.Is(err, device.ErrInUseDelete):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
