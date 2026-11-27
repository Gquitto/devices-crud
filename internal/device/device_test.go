package device_test

import (
	"errors"
	"testing"
	"uuid"

	"github.com/guilhermequitto/devices-crud/internal/device"
)

// Create
// Should create device obj, all fields OK.
func TestCreate_allFieldsOK(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateAvailable)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if d.ID == uuid.Nil() {
		t.Error("ID was not generated")
	}

	if d.Name != "DevTest" || d.Brand != "BrandTest" || d.State != device.StateAvailable {
		t.Errorf("Got %q/%q/%q, wanted DevTest/BrandTest/available", d.Name, d.Brand, d.State)
	}

	if d.CreationTime().IsZero() {
		t.Error("CreationTime was not generated")
	}
}

// Should generate a different ID for each device.
func TestCreate_diffUUIDValidation(t *testing.T) {
	d1, err := device.New("DevTest1", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	d2, err := device.New("DevTest2", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if d1.ID == d2.ID {
		t.Errorf("Id's should be different, %v", d1.ID)
	}
}

// Should default "state" to "available" when empty.
func TestCreate_defaultStateAvailable(t *testing.T) {
	d, err := device.New("DevTest1", "BrandTest", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if d.State != device.StateAvailable {
		t.Errorf("Default state should be 'available', got %q", d.State)
	}
}

// Should fail to create device, "state" invalid option.
func TestCreateFail_invalidState(t *testing.T) {
	d, err := device.New("DevTest1", "BrandTest", "anything-else")

	if !errors.Is(err, device.ErrInvalidState) {
		t.Fatalf("Expected ErrInvalidState, got %v", err)
	}

	if d != nil {
		t.Errorf("Expected no device, got %+v", d)
	}
}

// Should fail to create device, "name" blank ("" or whitespace-only).
func TestCreateFail_blankName(t *testing.T) {
	d, err := device.New("", "BrandTest", device.StateAvailable)

	if !errors.Is(err, device.ErrInvalidName) {
		t.Fatalf("Expected ErrInvalidName, got %v", err)
	}

	if d != nil {
		t.Errorf("Expected no device, got %+v", d)
	}
}

// Should fail to create device, "brand" blank ("" or whitespace-only).
func TestCreateFail_blankBrand(t *testing.T) {
	d, err := device.New("DevTest1", " ", device.StateAvailable)

	if !errors.Is(err, device.ErrInvalidBrand) {
		t.Fatalf("Expected ErrInvalidBrand, got %v", err)
	}

	if d != nil {
		t.Errorf("Expected no device, got %+v", d)
	}
}

// Edit
// Should fully edit device obj.
func TestEdit_allFieldsOK(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Update("DevTestUp", "BrandTestUp", device.StateInactive)
	if err != nil {
		t.Fatalf("Unexpected update error: %v", err)
	}

	if d.Name != "DevTestUp" || d.Brand != "BrandTestUp" || d.State != device.StateInactive {
		t.Errorf("Got %q/%q/%q, wanted DevTestUp/BrandTestUp/inactive", d.Name, d.Brand, d.State)
	}
}

// Should keep "id" and "creation time" unchanged after edit.
func TestEdit_sameIdAndCreationTime(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	d1ID := d.ID
	d1CT := d.CreationTime()

	err = d.Update("DevTestUp", "BrandTestUp", device.StateInactive)
	if err != nil {
		t.Fatalf("Unexpected update error: %v", err)
	}

	if d1ID != d.ID {
		t.Errorf("Non equal ID values D1: %v / D2: %v", d1ID, d.ID)
	}

	if !d1CT.Equal(d.CreationTime()) {
		t.Errorf("Non equal Creation Times D1: %v / D2: %v", d1CT, d.CreationTime())
	}
}

// Should edit considering default "state" to "available" when empty
func TestEdit_defaultStateAvailable(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateInactive)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Update("DevTestUp", "BrandTestUp", "")
	if err != nil {
		t.Fatalf("Unexpected update error: %v", err)
	}

	if d.State != device.StateAvailable {
		t.Errorf("Got %q, available", d.State)
	}
}

// Should fail to edit "state" invalid option.
func TestEditFail_invalidState(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Update("DevTestUp", "BrandTestUp", "anything-else")

	if !errors.Is(err, device.ErrInvalidState) {
		t.Fatalf("Expected ErrInvalidState, got %v", err)
	}

	if d.Name != "DevTest" {
		t.Errorf("Expected no changes, got %+v", d)
	}
}

// Should fail to edit "state" empty option when "in use".
func TestEditFail_invalidStateInUse(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateInUse)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Update("DevTest", "BrandTest", "")

	if !errors.Is(err, device.ErrInvalidState) {
		t.Fatalf("Expected ErrInvalidState, got %v", err)
	}

	if d.State != device.StateInUse {
		t.Error("Device should've stayed 'In Use'")
	}
}

// Should fail to edit, "name" or "brand" blank.
func TestEditFail_blankNameOrBrand(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Brand
	err = d.Update("DevTestUp", "", device.StateInactive)
	if !errors.Is(err, device.ErrInvalidBrand) {
		t.Fatalf("Expected ErrInvalidBrand, got %v", err)
	}
	if d.Brand != "BrandTest" {
		t.Errorf("Expected no changes, got %+v", d)
	}

	// Name
	err = d.Update("", "BrandTestUp", device.StateInactive)
	if !errors.Is(err, device.ErrInvalidName) {
		t.Fatalf("Expected ErrInvalidName, got %v", err)
	}
	if d.Name != "DevTest" {
		t.Errorf("Expected no changes, got %+v", d)
	}
}

// Should fail to edit "name" or "brand" if "state" is "in-use".
func TestEditFail_stateInUse(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateInUse)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	//
	err = d.Update("DevTestUp", "BrandTestUp", device.StateInactive)
	if !errors.Is(err, device.ErrInUseUpdate) {
		t.Fatalf("Expected ErrInUseUpdate, got %v", err)
	}
	if d.Brand != "BrandTest" {
		t.Errorf("Expected no changes, got %+v", d)
	}
}

// Should edit only "state" of an "in-use" device (e.g. in-use -> available).
func TestEdit_stateInUse(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateInUse)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Update("DevTest", "BrandTest", device.StateAvailable)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if d.State != device.StateAvailable {
		t.Error("Error while updating state")
	}
}

// Delete
// Should allow deleting an "available" or "inactive" device.
func TestDelete_deleteOk(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateAvailable)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Delete()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// Should not allow deleting an "in-use" device.
func TestDeleteFail_stateInUse(t *testing.T) {
	d, err := device.New("DevTest", "BrandTest", device.StateInUse)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = d.Delete()

	if !errors.Is(err, device.ErrInUseDelete) {
		t.Fatalf("Expected ErrInUseDelete, got %v", err)
	}
}
