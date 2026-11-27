package device

import (
	"errors"
	"strings"
	"time"
	"uuid"
)

// State is the lifecycle state of a device.
type State string

// Allowed device states.
const (
	StateAvailable State = "available"
	StateInUse     State = "in-use"
	StateInactive  State = "inactive"
)

// Domain errors.
var (
	// ErrInvalidState is returned when a state is not one of the allowed
	// values, or when an in-use device is updated with an empty state.
	ErrInvalidState = errors.New("invalid device state")
	// ErrInvalidName is returned when the name is empty or whitespace-only.
	ErrInvalidName = errors.New("device name must not be blank")
	// ErrInvalidBrand is returned when the brand is empty or whitespace-only.
	ErrInvalidBrand = errors.New("device brand must not be blank")
	// ErrInUseUpdate is returned when name or brand of an in-use device change.
	ErrInUseUpdate = errors.New("cannot update name or brand of an in-use device")
	// ErrInUseDelete is returned when deleting an in-use device.
	ErrInUseDelete = errors.New("cannot delete an in-use device")
)

// IsValid reports whether s is one of the allowed states.
func (s State) IsValid() bool {
	switch s {
	case StateAvailable, StateInUse, StateInactive:
		return true
	}
	return false
}

// Device is the core domain entity.
//
// creationTime is unexported on purpose: it is set only by New and is
// read through CreationTime, so no caller can change it.
type Device struct {
	ID    uuid.UUID
	Name  string
	Brand string
	State State

	creationTime time.Time
}

// New creates a device with a fresh ID and the current time as its
// creation time. An empty state defaults to StateAvailable.
func New(name, brand string, state State) (*Device, error) {
	if err := validateNameBrand(name, brand); err != nil {
		return nil, err
	}
	state, err := resolveState(state)
	if err != nil {
		return nil, err
	}
	return &Device{
		ID:           uuid.New(),
		Name:         name,
		Brand:        brand,
		State:        state,
		creationTime: time.Now().UTC(),
	}, nil
}

// CreationTime returns when the device was created.
func (d *Device) CreationTime() time.Time {
	return d.creationTime
}

// Update replaces name, brand and state. All rules are checked before
// anything is applied, so on error the device is left unchanged.
//
// An empty state defaults to StateAvailable, except when the device is
// currently in use: it is never freed implicitly. Name and brand cannot
// change while the device is in use; the rule is evaluated against the
// current state, before the update.
func (d *Device) Update(name, brand string, state State) error {
	if err := validateNameBrand(name, brand); err != nil {
		return err
	}
	if state == "" && d.State == StateInUse {
		return ErrInvalidState
	}
	state, err := resolveState(state)
	if err != nil {
		return err
	}
	if d.State == StateInUse && (name != d.Name || brand != d.Brand) {
		return ErrInUseUpdate
	}

	d.Name = name
	d.Brand = brand
	d.State = state
	return nil
}

// Delete checks whether the device may be deleted. It does not remove
// anything: removal is the repository's job once this returns nil.
func (d *Device) Delete() error {
	if d.State == StateInUse {
		return ErrInUseDelete
	}
	return nil
}

func validateNameBrand(name, brand string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(brand) == "" {
		return ErrInvalidBrand
	}
	return nil
}

// resolveState applies the empty-state default and validates the result.
func resolveState(s State) (State, error) {
	if s == "" {
		return StateAvailable, nil
	}
	if !s.IsValid() {
		return "", ErrInvalidState
	}
	return s, nil
}
