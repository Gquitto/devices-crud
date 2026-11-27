package device

import (
	"context"
	"errors"
	"uuid"
)

// ErrNotFound is returned when no device has the requested id.
var ErrNotFound = errors.New("device not found")

// Filter narrows a device listing. Zero-value fields are ignored, so an
// empty Filter matches every device. Name and Brand are case-insensitive
// partial matches; State is an exact match. Fields combine with AND.
type Filter struct {
	Name  string
	Brand string
	State State
}

// Repository persists devices. Implementations return ErrNotFound from
// Get, Update and Delete when the id does not exist. They store data only:
// domain rules are enforced by Device and Service.
type Repository interface {
	Create(ctx context.Context, d *Device) error
	Get(ctx context.Context, id uuid.UUID) (*Device, error)
	List(ctx context.Context, f Filter) ([]Device, error)
	Update(ctx context.Context, d *Device) error
	Delete(ctx context.Context, id uuid.UUID) error
}
