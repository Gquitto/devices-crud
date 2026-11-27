package sqlite_test

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"uuid"

	"github.com/guilhermequitto/devices-crud/internal/device"
	"github.com/guilhermequitto/devices-crud/internal/storage/sqlite"
)

//Setup

// openRepo opens (or reopens) the database at path.
func openRepo(t *testing.T, path string) *sqlite.Repository {
	t.Helper()
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return sqlite.NewRepository(db)
}

// newRepo opens a fresh database in a temp dir.
func newRepo(t *testing.T) *sqlite.Repository {
	t.Helper()
	return openRepo(t, filepath.Join(t.TempDir(), "test.db"))
}

// mustCreate builds a valid device and stores it.
func mustCreate(t *testing.T, repo *sqlite.Repository, name, brand string, state device.State) *device.Device {
	t.Helper()
	d, err := device.New(name, brand, state)
	if err != nil {
		t.Fatalf("new device: %v", err)
	}
	if err := repo.Create(t.Context(), d); err != nil {
		t.Fatalf("create: %v", err)
	}
	return d
}

// assertSame compares field by field: time.Time must be compared with Equal,
// because == also compares the location, which a DB round trip can change.
func assertSame(t *testing.T, got, want *device.Device) {
	t.Helper()
	if got.ID != want.ID || got.Name != want.Name || got.Brand != want.Brand || got.State != want.State {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if !got.CreationTime().Equal(want.CreationTime()) {
		t.Errorf("creationTime %v, want %v", got.CreationTime(), want.CreationTime())
	}
}

// /////////////////////////////////////////////////
// Should read back stored, creationTime included.
func TestRepoCreateGet_roundTrip(t *testing.T) {
	repo := newRepo(t)
	d := mustCreate(t, repo, "TestDevice", "TestBrand", device.StateInUse)

	got, err := repo.Get(t.Context(), d.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	assertSame(t, got, d)
}

// Should persist name, brand and state changes and keep creationTime.
func TestRepoUpdate_persistsChanges(t *testing.T) {
	repo := newRepo(t)
	d := mustCreate(t, repo, "TestDevice", "TestBrand", device.StateAvailable)

	if err := d.Update("iPhone 16", "TestBrand Inc", device.StateInactive); err != nil {
		t.Fatalf("domain update: %v", err)
	}
	if err := repo.Update(t.Context(), d); err != nil {
		t.Fatalf("repo update: %v", err)
	}

	got, err := repo.Get(t.Context(), d.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	assertSame(t, got, d)
}

// Should remove the device.
func TestRepoDelete_removesDevice(t *testing.T) {
	repo := newRepo(t)
	d := mustCreate(t, repo, "TestDevice", "TestBrand", device.StateAvailable)

	if err := repo.Delete(t.Context(), d.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(t.Context(), d.ID); !errors.Is(err, device.ErrNotFound) {
		t.Errorf("get after delete: %v, want ErrNotFound", err)
	}
}

// Should return ErrNotFound for an unknown id on Get, Update and Delete.
func TestRepoFail_notFound(t *testing.T) {
	repo := newRepo(t)
	unsaved, _ := device.New("TestDevice", "TestBrand", device.StateAvailable) // never stored

	if _, err := repo.Get(t.Context(), uuid.New()); !errors.Is(err, device.ErrNotFound) {
		t.Errorf("Got: %v, want ErrNotFound", err)
	}
	if err := repo.Update(t.Context(), unsaved); !errors.Is(err, device.ErrNotFound) {
		t.Errorf("Update: %v, want ErrNotFound", err)
	}
	if err := repo.Delete(t.Context(), uuid.New()); !errors.Is(err, device.ErrNotFound) {
		t.Errorf("Delete: %v, want ErrNotFound", err)
	}
}

// Should filter by name/brand (partial, case-insensitive) and state (exact), combined with AND.
func TestRepoList_filters(t *testing.T) {
	repo := newRepo(t)
	mustCreate(t, repo, "TestDevice", "TestBrand1", device.StateAvailable)
	mustCreate(t, repo, "promax", "TestBrand2", device.StateInUse)
	mustCreate(t, repo, "RogAllyX", "TestBrand1", device.StateInactive)

	cases := []struct {
		name   string
		filter device.Filter
		want   []string // sorted device names
	}{
		{"no filter", device.Filter{}, []string{"RogAllyX", "TestDevice", "promax"}},
		{"brand partial, case-insensitive", device.Filter{Brand: "brand1"}, []string{"RogAllyX", "TestDevice"}},
		{"name partial, case-insensitive", device.Filter{Name: "PROMAX"}, []string{"promax"}},
		{"state exact", device.Filter{State: device.StateInUse}, []string{"promax"}},
		{"combined with AND", device.Filter{Brand: "brand1", State: device.StateInactive}, []string{"RogAllyX"}},
		{"no match", device.Filter{Brand: "nokia"}, []string{}},
	}
	for _, tcase := range cases {
		t.Run(tcase.name, func(t *testing.T) {
			list, err := repo.List(t.Context(), tcase.filter)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			names := []string{}
			for _, d := range list {
				names = append(names, d.Name)
			}
			slices.Sort(names)
			if !slices.Equal(names, tcase.want) {
				t.Errorf("got %v, want %v", names, tcase.want)
			}
		})
	}
}
