CREATE TABLE devices (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    brand         TEXT NOT NULL,
    state         TEXT NOT NULL CHECK (state IN ('available', 'in-use', 'inactive')),
    creation_time TEXT NOT NULL -- RFC 3339 with nanoseconds, UTC
);

CREATE INDEX idx_devices_state ON devices (state);
