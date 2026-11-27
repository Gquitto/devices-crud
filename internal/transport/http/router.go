package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"uuid"

	"github.com/gin-gonic/gin"

	"github.com/guilhermequitto/devices-crud/internal/device"
)

// NewRouter returns the HTTP handler for the API described in
// api/openapi.yaml, backed by svc.
func NewRouter(svc *device.Service) http.Handler {
	h := &handler{svc: svc}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	devices := r.Group("/devices")
	devices.POST("", h.create)
	devices.GET("", h.list)
	devices.GET("/:id", h.get)
	devices.PUT("/:id", h.update)
	devices.PATCH("/:id", h.patch)
	devices.DELETE("/:id", h.delete)

	return r
}

type handler struct {
	svc *device.Service
}

func (h *handler) create(c *gin.Context) {
	var req writeRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}
	d, err := h.svc.Create(c.Request.Context(), req.Name, req.Brand, req.State)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Location", "/devices/"+d.ID.String())
	c.JSON(http.StatusCreated, toResponse(d))
}

func (h *handler) list(c *gin.Context) {
	list, err := h.svc.Filter(c.Request.Context(),
		c.Query("name"), c.Query("brand"), device.State(c.Query("state")))
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]deviceResponse, 0, len(list)) // never null in JSON
	for i := range list {
		out = append(out, toResponse(&list[i]))
	}
	c.JSON(http.StatusOK, out)
}

func (h *handler) get(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	d, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(d))
}

func (h *handler) update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var req writeRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}
	d, err := h.svc.Update(c.Request.Context(), id, req.Name, req.Brand, req.State)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(d))
}

func (h *handler) patch(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var req patchRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}
	d, err := h.svc.Patch(c.Request.Context(), id, req.toPatch())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(d))
}

func (h *handler) delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// pathID parses the :id path parameter.
func pathID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%w: invalid device id", errBadRequest)
	}
	return id, nil
}

// decodeJSON reads exactly one JSON object into v, rejecting unknown
// fields (so id and creationTime cannot be sent) and trailing data.
func decodeJSON(c *gin.Context, v any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: invalid JSON body: %v", errBadRequest, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: body must contain a single JSON object", errBadRequest)
	}
	return nil
}
