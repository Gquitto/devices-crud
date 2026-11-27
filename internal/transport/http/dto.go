package httpapi

import (
	"time"

	"github.com/guilhermequitto/devices-crud/internal/device"
)

// deviceResponse is the Device schema in api/openapi.yaml.
type deviceResponse struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Brand        string       `json:"brand"`
	State        device.State `json:"state"`
	CreationTime time.Time    `json:"creationTime"`
}

func toResponse(d *device.Device) deviceResponse {
	return deviceResponse{
		ID:           d.ID.String(),
		Name:         d.Name,
		Brand:        d.Brand,
		State:        d.State,
		CreationTime: d.CreationTime(),
	}
}

// writeRequest is the body of POST and PUT. A missing name or brand decodes
// to "" and is rejected by the domain; a missing state decodes to "" and
// gets the domain default.
type writeRequest struct {
	Name  string       `json:"name"`
	Brand string       `json:"brand"`
	State device.State `json:"state"`
}

// patchRequest is the body of PATCH. Pointers tell an omitted field (nil)
// from one sent empty, so it maps one-to-one onto device.Patch.
type patchRequest struct {
	Name  *string       `json:"name"`
	Brand *string       `json:"brand"`
	State *device.State `json:"state"`
}

func (p patchRequest) toPatch() device.Patch {
	return device.Patch{Name: p.Name, Brand: p.Brand, State: p.State}
}

type errorResponse struct {
	Error string `json:"error"`
}
