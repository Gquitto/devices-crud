// Package httpapi exposes the device use cases over HTTP using gin: routing,
// request/response DTOs, input binding and mapping of domain errors to
// HTTP status codes. The contract is documented in api/openapi.yaml.
//
// The package lives in internal/transport/http but is named httpapi so it
// does not clash with net/http.
package httpapi
