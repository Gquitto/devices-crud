package device

import (
	"context"
	"errors"
	"uuid"
)

// ErrInvalidPatch is returned by Service.Patch when no field is set.
var ErrInvalidPatch = errors.New("patch must set at least one field")

// Patch holds the fields of a partial update. A nil field is omitted and
// keeps its current value; a non-nil field is applied and validated like
// any update, so a pointer to "" is a blank value, not an omission.
type Patch struct {
	Name  *string
	Brand *string
	State *State
}

func (p Patch) isEmpty() bool {
	return p.Name == nil && p.Brand == nil && p.State == nil
}

// Service runs device use cases: it loads devices from the repository,
// applies the domain rules and persists the result.
type Service struct {
	repo Repository
}

// NewService returns a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create validates and persists a new device.
func (s *Service) Create(ctx context.Context, name, brand string, state State) (*Device, error) {
	d, err := New(name, brand, state)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Get returns the device with the given id, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Device, error) {
	return s.repo.Get(ctx, id)
}

// Filter lists devices matching name, brand and state. Empty arguments are
// ignored, so Filter(ctx, "", "", "") returns every device. A non-empty
// state that is not a valid State returns ErrInvalidState.
func (s *Service) Filter(ctx context.Context, name, brand string, state State) ([]Device, error) {
	if state != "" && !state.IsValid() {
		return nil, ErrInvalidState
	}
	return s.repo.List(ctx, Filter{Name: name, Brand: brand, State: state})
}

// Update replaces name, brand and state of an existing device. Nothing is
// persisted if a domain rule fails.
func (s *Service) Update(ctx context.Context, id uuid.UUID, name, brand string, state State) (*Device, error) {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.update(ctx, d, name, brand, state)
}

// update applies the domain update to d and persists it. d must have been
// loaded from the repository; on a rule failure nothing is stored.
func (s *Service) update(ctx context.Context, d *Device, name, brand string, state State) (*Device, error) {
	if err := d.Update(name, brand, state); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Patch changes only the fields set in p; omitted fields keep their current
// value. An empty patch returns ErrInvalidPatch. Nothing is persisted if a
// domain rule fails.
func (s *Service) Patch(ctx context.Context, id uuid.UUID, p Patch) (*Device, error) {
	if p.isEmpty() {
		return nil, ErrInvalidPatch
	}
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	name, brand, state := d.Name, d.Brand, d.State
	if p.Name != nil {
		name = *p.Name
	}
	if p.Brand != nil {
		brand = *p.Brand
	}
	if p.State != nil {
		state = *p.State
	}

	return s.update(ctx, d, name, brand, state)
}

// Delete removes a device. In-use devices cannot be deleted.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	d, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := d.Delete(); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
