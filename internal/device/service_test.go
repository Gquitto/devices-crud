package device_test

import (
	"errors"
	"os"
	"testing"
	"uuid"

	"github.com/guilhermequitto/devices-crud/internal/device"
	"github.com/guilhermequitto/devices-crud/internal/device/devicetest"
)

// Setup specific cases
var (
	seededRepo      *devicetest.FakeRepo
	idAllOk         = uuid.MustParse("6f4129d1-4348-42d4-ad60-56766b0f969a")
	idToPutOk       = uuid.MustParse("a76e4283-693e-463b-9aae-b9ed92694506")
	idToPutInUse    = uuid.MustParse("652a5ccd-989c-4d68-bad6-b9096f251c5e")
	idToPatch       = uuid.MustParse("3b1f6c2e-8d4a-4f7e-9c11-2a5d7e8f9b01")
	idToPatchEmpty  = uuid.MustParse("9e2d4a6b-1c3f-4b8e-a7d2-5f6e8c9a0b12")
	idToDelete      = uuid.MustParse("c4a8e1f2-6b3d-4e9a-8f7c-1d2e3f4a5b23")
	idToDeleteInUse = uuid.MustParse("e7f1a2b3-4c5d-4e6f-9a8b-7c6d5e4f3a34")
)

func TestMain(m *testing.M) {
	seededRepo = devicetest.NewFakeRepo()
	// Seeds
	seed(device.StateAvailable, idAllOk, idToPutOk, idToPatchEmpty, idToDelete)
	seed(device.StateInUse, idToPutInUse, idToDeleteInUse)
	seed(device.StateInactive, idToPatch)

	os.Exit(m.Run()) // Runs tests in package
}

func seed(state device.State, ids ...uuid.UUID) {
	d, err := device.New("DevTest", "BrandTest", state)
	if err != nil {
		panic(err)
	}
	for _, id := range ids {
		d.ID = id
		seededRepo.Devices[id] = *d // *d copies the struct, so each entry is independent
	}
}

// Create
// Should create and persist a device.
func TestServiceCreate_saveDevice(t *testing.T) {
	repo := devicetest.NewFakeRepo()
	svc := device.NewService(repo)

	got, err := svc.Create(t.Context(), "DevTest", "BrandTest", device.StateAvailable)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stored, ok := repo.Devices[got.ID]
	if !ok {
		t.Fatalf("Device %v was not persisted", got.ID)
	}
	if stored != *got {
		t.Errorf("Stored device %+v differs from returned %+v", stored, *got)
	}
}

// Should not persist when domain validation fails.
func TestServiceCreateFail_domainValidation(t *testing.T) {
	repo := devicetest.NewFakeRepo()
	svc := device.NewService(repo)

	_, err := svc.Create(t.Context(), "", "BrandTest", device.StateAvailable)
	if !errors.Is(err, device.ErrInvalidName) {
		t.Errorf("Expected ErrInvalidName, got error: %v", err)
	}

	if len(repo.Devices) != 0 {
		t.Errorf("Expected empty repo, got %d device(s)", len(repo.Devices))
	}
}

// Get
// Should get a device by id.
func TestServiceGet_byId(t *testing.T) {
	svc := device.NewService(seededRepo)

	d, err := svc.Get(t.Context(), idAllOk)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if d.ID != idAllOk {
		t.Errorf("Item retrievied not the same: %v", d.ID)
	}
}

// Should fail with ErrNotFound when id does not exist.
func TestServiceGetFail_byId(t *testing.T) {
	repo := devicetest.NewFakeRepo()
	svc := device.NewService(repo)

	d, err := svc.Get(t.Context(), uuid.New())
	if !errors.Is(err, device.ErrNotFound) {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
	}

	if d != nil {
		t.Errorf("Item retrievied incorrectly: %v", d.ID)
	}
}

// List
// Should fetch all devices.
func TestServiceList_fetchAll(t *testing.T) {
	svc := device.NewService(seededRepo)

	dList, err := svc.Filter(t.Context(), "", "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(dList) < 1 {
		t.Error("Didn't fetch nothing man")
	}
}

// Should fail to list with an invalid state filter (ErrInvalidState).
func TestServiceListFail_invalidStateSearch(t *testing.T) {
	repo := devicetest.NewFakeRepo()
	svc := device.NewService(repo)

	_, err := svc.Filter(t.Context(), "", "", "anything-else")
	if !errors.Is(err, device.ErrInvalidState) {
		t.Errorf("Expected ErrInvalidState, got: %v", err)
	}
}

// Full Update
// Should fully update and persist a device.
func TestServiceUpdate_persistDevice(t *testing.T) {
	svc := device.NewService(seededRepo)

	got, err := svc.Update(t.Context(), idToPutOk, "DevTestUp", "BrandTestUp", device.StateInactive)
	if err != nil {
		t.Fatalf("Unexpected update error: %v", err)
	}

	stored := seededRepo.Devices[idToPutOk]
	if stored.Name != "DevTestUp" {
		t.Errorf("Stored %q, wanted DevTestUp", stored.Name)
	}
	if stored != *got {
		t.Errorf("Stored device %+v differs from returned %+v", stored, *got)
	}
}

// Should not persist when a domain rule fails (stored device unchanged).
func TestServiceUpdateFail_domainRule(t *testing.T) {
	svc := device.NewService(seededRepo)

	_, err := svc.Update(t.Context(), idToPutInUse, "DevTestUp", "BrandTestUp", device.StateInactive)
	if !errors.Is(err, device.ErrInUseUpdate) {
		t.Fatalf("Expected ErrInUseUpdate, got: %v", err)
	}

	stored := seededRepo.Devices[idToPutInUse]
	if stored.Name == "DevTestUp" {
		t.Errorf("Stored device changed! %+v", stored)
	}
}

// Should fail with ErrNotFound when id does not exist.
func TestServiceUpdateFail_notFound(t *testing.T) {
	repo := devicetest.NewFakeRepo()
	svc := device.NewService(repo)

	_, err := svc.Update(t.Context(), uuid.New(), "DevTestUp", "BrandTestUp", device.StateAvailable)
	if !errors.Is(err, device.ErrNotFound) {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
	}

	if len(repo.Devices) != 0 {
		t.Errorf("Expected empty repo, got %d device(s)", len(repo.Devices))
	}
}

// Partial Update //Todo
// Should change only the provided fields (omitted ones)
func TestServicePatch_providedFields(t *testing.T) {
	svc := device.NewService(seededRepo)

	_, err := svc.Patch(t.Context(), idToPatch, device.Patch{Name: new("DevTestUp"), State: new(device.StateAvailable)})
	if err != nil {
		t.Fatalf("Unexpected patch error: %v", err)
	}

	d := seededRepo.Devices[idToPatch]
	if d.Name != "DevTestUp" || d.Brand != "BrandTest" || d.State != device.StateAvailable {
		t.Errorf("Patch fail: %+v", d)
	}

}

// Should fail to patch with no fields (nothing persisted).
func TestServicePatchFail_noFields(t *testing.T) {
	svc := device.NewService(seededRepo)

	_, err := svc.Patch(t.Context(), idToPatchEmpty, device.Patch{})
	if !errors.Is(err, device.ErrInvalidPatch) {
		t.Fatalf("Expected ErrInvalidPatch, got: %v", err)
	}

	d := seededRepo.Devices[idToPatchEmpty]
	if d.Name != "DevTest" || d.State != device.StateAvailable {
		t.Errorf("Patch should not have succeded: %+v", d)
	}

}

// Delete
// Should delete a device (no longer in repo).
func TestServiceDelete_removeDevice(t *testing.T) {
	svc := device.NewService(seededRepo)

	err := svc.Delete(t.Context(), idToDelete)
	if err != nil {
		t.Fatalf("Unexpected delete error: %v", err)
	}

	if _, ok := seededRepo.Devices[idToDelete]; ok {
		t.Error("Deleted device should no longer be saved")
	}
}

// Should fail to delete an "in-use" device (still in repo).
func TestServiceDeleteFail_inUseDevice(t *testing.T) {
	svc := device.NewService(seededRepo)

	err := svc.Delete(t.Context(), idToDeleteInUse)
	if !errors.Is(err, device.ErrInUseDelete) {
		t.Fatalf("Expected ErrInUseDelete, got: %v", err)
	}

	if _, ok := seededRepo.Devices[idToDeleteInUse]; !ok {
		t.Error("Device should not be deleted")
	}
}

// Should fail with ErrNotFound when id does not exist.
func TestServiceDeleteFail_notFound(t *testing.T) {
	svc := device.NewService(seededRepo)

	err := svc.Delete(t.Context(), uuid.New())
	if !errors.Is(err, device.ErrNotFound) {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
	}
}
