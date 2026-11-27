// Package devicetest provides test doubles for the device package, in the
// spirit of net/http/httptest.
//
// # FakeRepo
//
// FakeRepo is an in-memory device.Repository for service tests. It is a
// plain map with no logic of its own, so a failing test points at the
// service, not at the fake.
//
// Setup and inspection go straight through the exported Devices map:
//
//	repo := devicetest.NewFakeRepo()
//	svc := device.NewService(repo)
//
//	// seed a device, bypassing the service
//	d, _ := device.New("iPhone 15", "Apple", device.StateInUse)
//	repo.Devices[d.ID] = *d
//
//	// ... call svc ...
//
//	// assert on what was persisted
//	stored, ok := repo.Devices[d.ID]
//
// Behavior:
//   - Devices are stored by value. A *Device returned by Get is a copy,
//     so changes reach the map only through Update. This lets tests check
//     that a failed operation persisted nothing.
//   - Get, Update and Delete return device.ErrNotFound for an unknown id.
//   - Create overwrites an existing id without error.
//   - List ignores the Filter and returns every device in random order.
//     Filtering happens in SQL and is tested in internal/storage/sqlite.
//   - No domain rules are enforced; the service is responsible for them.
//   - Not safe for concurrent use. Do not share one FakeRepo between
//     parallel tests.
//
// # Ownership
//
// Unlike *_test.go files, this package is written by the agent. It must stay
// a faithful, logic-free implementation of device.Repository and must never
// be changed to make a test pass. See CLAUDE.md.
package devicetest
