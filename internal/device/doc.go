// Package device holds the Device domain: the entity, its states, the
// domain validation rules and the service that orchestrates use cases.
//
// This package must not depend on HTTP or storage details (no gin, no SQL).
// It defines the repository interface that storage implementations satisfy.
package device
