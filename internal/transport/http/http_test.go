package httpapi_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/guilhermequitto/devices-crud/internal/device"
	"github.com/guilhermequitto/devices-crud/internal/storage/sqlite"
	httpapi "github.com/guilhermequitto/devices-crud/internal/transport/http"
)

// deviceJSON mirrors the Device schema in api/openapi.yaml.
type deviceJSON struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Brand        string    `json:"brand"`
	State        string    `json:"state"`
	CreationTime time.Time `json:"creationTime"`
}

// ---- helpers ----

// newRouter builds the full stack on a fresh database in a temp dir.
func newRouter(t *testing.T) http.Handler {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return httpapi.NewRouter(device.NewService(sqlite.NewRepository(db)))
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func create(t *testing.T, h http.Handler, name, brand string, state device.State) deviceJSON {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"brand":%q,"state":%q}`, name, brand, state)
	rec := do(t, h, http.MethodPost, "/devices", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body)
	}
	return decode[deviceJSON](t, rec)
}

// HTTPs
// Should answer the health check 200.
func TestHealth(t *testing.T) {
	h := newRouter(t)

	if rec := do(t, h, http.MethodGet, "/health", ""); rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rec.Code)
	}
}

// Should return the gotten device as JSON.
func TestHTTPs_returnsDevice(t *testing.T) {
	h := newRouter(t)
	created := create(t, h, "14proplus", "Realme", device.StateInUse)

	rec := do(t, h, http.MethodGet, "/devices/"+created.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}

	got := decode[deviceJSON](t, rec)
	if got.ID != created.ID {
		t.Errorf("got %+v, want %+v", got, created)
	}
	if !got.CreationTime.Equal(created.CreationTime) {
		t.Errorf("creationTime %v, want %v", got.CreationTime, created.CreationTime)
	}
}

// Should return 204 on Delete and 404 on Get.
func TestHTTPs_notFound(t *testing.T) {
	h := newRouter(t)
	created := create(t, h, "14proplus", "Realme", device.StateAvailable)
	path := "/devices/" + created.ID

	if rec := do(t, h, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status %d, want 204", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, path, ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET after delete: status %d, want 404", rec.Code)
	}
}

// Should pass name, brand and state query params to the filter.
func TestHTTPs_queryParams(t *testing.T) {
	h := newRouter(t)
	create(t, h, "14proplus", "Realme", device.StateInUse)
	create(t, h, "14proplus", "Realme", device.StateAvailable)
	create(t, h, "Yoga Slim", "Lenovo", device.StateInUse)

	rec := do(t, h, http.MethodGet, "/devices?name=proplus&brand=realme&state=in-use", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	got := decode[[]deviceJSON](t, rec)
	if len(got) != 1 || got[0].Name != "14proplus" || got[0].State != "in-use" {
		t.Errorf("got %+v, want only the in-use 14proplus", got)
	}
}

// Should return 200 and empty when nothing matches.
func TestHTTPs_emptyIsArray(t *testing.T) {
	h := newRouter(t)

	rec := do(t, h, http.MethodGet, "/devices?brand=nokia", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body %q, want []", body)
	}
}

// Should reject invalid input with 400 and an error body.
func TestHTTPs_badRequest(t *testing.T) {
	h := newRouter(t)
	avail := create(t, h, "14proplus", "Realme", device.StateAvailable)
	inUse := create(t, h, "Yoga Slim", "Lenovo", device.StateInUse)

	cases := []struct {
		name, method, path, body string
	}{
		{"malformed JSON", http.MethodPost, "/devices", `{"name":`},
		{"unknown field id", http.MethodPost, "/devices", `{"id":"x","name":"A","brand":"B"}`},
		{"unknown field creationTime", http.MethodPut, "/devices/" + avail.ID, `{"name":"A","brand":"B","creationTime":"2020-01-01T00:00:00Z"}`},
		{"invalid id in path", http.MethodGet, "/devices/not-a-uuid", ""},
		{"invalid state filter", http.MethodGet, "/devices?state=xyz", ""},
		{"empty patch", http.MethodPatch, "/devices/" + avail.ID, `{}`},
		{"blank name on patch", http.MethodPatch, "/devices/" + avail.ID, `{"name":""}`},
		{"put without state on in-use", http.MethodPut, "/devices/" + inUse.ID, `{"name":"Yoga Slim","brand":"Lenovo"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want 400, body %s", rec.Code, rec.Body)
			}
			if e := decode[map[string]any](t, rec); e["error"] == nil {
				t.Errorf("missing error message: %s", rec.Body)
			}
		})
	}
}

// Should return 404 for an unknown id on every method.
func TestHTTPs_notFoundID(t *testing.T) {
	h := newRouter(t)
	path := "/devices/" + uuid.New().String()

	cases := []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPut, `{"name":"A","brand":"B","state":"available"}`},
		{http.MethodPatch, `{"name":"A"}`},
		{http.MethodDelete, ""},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			if rec := do(t, h, tc.method, path, tc.body); rec.Code != http.StatusNotFound {
				t.Errorf("status %d, want 404, body %s", rec.Code, rec.Body)
			}
		})
	}
}

// Should return 409 when a domain rule blocks the operation on an in-use device.
func TestConflict(t *testing.T) {
	h := newRouter(t)
	inUse := create(t, h, "Yoga Slim", "Lenovo", device.StateInUse)
	path := "/devices/" + inUse.ID

	cases := []struct{ name, method, body string }{
		{"put renaming in-use", http.MethodPut, `{"name":"Other","brand":"Lenovo","state":"in-use"}`},
		{"patch renaming in-use", http.MethodPatch, `{"name":"Other"}`},
		{"delete in-use", http.MethodDelete, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := do(t, h, tc.method, path, tc.body); rec.Code != http.StatusConflict {
				t.Errorf("status %d, want 409, body %s", rec.Code, rec.Body)
			}
		})
	}
}
