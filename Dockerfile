# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && mkdir -p /out/data

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/api /app/api
# Pre-create the data dir owned by the nonroot user so a named volume mounted
# here inherits writable permissions.
COPY --from=build --chown=65532:65532 /out/data /data

ENV GIN_MODE=release \
    PORT=8080 \
    DB_PATH=/data/devices.db

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
