package devicetest

import (
	"context"
	"uuid"

	"github.com/guilhermequitto/devices-crud/internal/device"
)

// FakeRepo is an in-memory device.Repository. See the package documentation
// for its exact behavior.
type FakeRepo struct {
	// Devices is the fake's storage, keyed by device id. Tests may read and
	// write it directly to seed data and to assert on what was persisted.
	Devices map[uuid.UUID]device.Device
}

// Compile-time check that FakeRepo satisfies device.Repository.
var _ device.Repository = (*FakeRepo)(nil)

// NewFakeRepo returns an empty FakeRepo.
func NewFakeRepo() *FakeRepo {
	return &FakeRepo{Devices: map[uuid.UUID]device.Device{}}
}

// Create stores a copy of d, overwriting any device with the same id.
func (r *FakeRepo) Create(_ context.Context, d *device.Device) error {
	r.Devices[d.ID] = *d
	return nil
}

// Get returns a copy of the stored device, or device.ErrNotFound.
func (r *FakeRepo) Get(_ context.Context, id uuid.UUID) (*device.Device, error) {
	d, ok := r.Devices[id]
	if !ok {
		return nil, device.ErrNotFound
	}
	return &d, nil
}

// List returns copies of all stored devices in random order. The filter is
// ignored: filtering is the SQL repository's job and is tested there.
func (r *FakeRepo) List(_ context.Context, _ device.Filter) ([]device.Device, error) {
	out := make([]device.Device, 0, len(r.Devices))
	for _, d := range r.Devices {
		out = append(out, d)
	}
	return out, nil
}

// Update replaces the stored device with a copy of d, or returns
// device.ErrNotFound if no device has d's id.
func (r *FakeRepo) Update(_ context.Context, d *device.Device) error {
	if _, ok := r.Devices[d.ID]; !ok {
		return device.ErrNotFound
	}
	r.Devices[d.ID] = *d
	return nil
}

// Delete removes the device, or returns device.ErrNotFound.
func (r *FakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.Devices[id]; !ok {
		return device.ErrNotFound
	}
	delete(r.Devices, id)
	return nil
}
