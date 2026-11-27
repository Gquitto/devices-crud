package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/guilhermequitto/devices-crud/internal/device"
)

// Repository is a device.Repository backed by SQLite.
type Repository struct {
	db *sql.DB
}

// Compile-time check that Repository satisfies device.Repository.
var _ device.Repository = (*Repository)(nil)

// NewRepository returns a Repository using db, which must have been opened
// with Open so the schema exists.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// creationTime is stored as RFC 3339 text with nanoseconds in UTC, so it
// reads back exactly and sorts chronologically as a string.
const timeLayout = time.RFC3339Nano

// Create inserts d.
func (r *Repository) Create(ctx context.Context, d *device.Device) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (id, name, brand, state, creation_time) VALUES (?, ?, ?, ?, ?)`,
		d.ID.String(), d.Name, d.Brand, string(d.State), d.CreationTime().UTC().Format(timeLayout))
	if err != nil {
		return fmt.Errorf("insert device: %w", err)
	}
	return nil
}

// Get returns the device with the given id, or device.ErrNotFound.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*device.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, brand, state, creation_time FROM devices WHERE id = ?`, id.String())
	d, err := scanDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, device.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	return d, nil
}

// List returns the devices matching f, oldest first. Name and brand are
// case-insensitive substring matches; state is exact. Empty fields are
// ignored. The result is never nil.
func (r *Repository) List(ctx context.Context, f device.Filter) ([]device.Device, error) {
	var (
		where []string
		args  []any
	)
	// instr is used instead of LIKE so % and _ in the input match literally.
	if f.Name != "" {
		where = append(where, `instr(lower(name), lower(?)) > 0`)
		args = append(args, f.Name)
	}
	if f.Brand != "" {
		where = append(where, `instr(lower(brand), lower(?)) > 0`)
		args = append(args, f.Brand)
	}
	if f.State != "" {
		where = append(where, `state = ?`)
		args = append(args, string(f.State))
	}

	query := `SELECT id, name, brand, state, creation_time FROM devices`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY creation_time, id`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	out := []device.Device{}
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("list devices: %w", err)
		}
		out = append(out, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return out, nil
}

// Update stores d's name, brand and state, or returns device.ErrNotFound.
// creation_time is never written after insert.
func (r *Repository) Update(ctx context.Context, d *device.Device) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE devices SET name = ?, brand = ?, state = ? WHERE id = ?`,
		d.Name, d.Brand, string(d.State), d.ID.String())
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}
	return requireOneRow(res)
}

// Delete removes the device, or returns device.ErrNotFound.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM devices WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	return requireOneRow(res)
}

func requireOneRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return device.ErrNotFound
	}
	return nil
}

// scanner is satisfied by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanDevice(s scanner) (*device.Device, error) {
	var id, name, brand, state, created string
	if err := s.Scan(&id, &name, &brand, &state, &created); err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse id %q: %w", id, err)
	}
	ct, err := time.Parse(timeLayout, created)
	if err != nil {
		return nil, fmt.Errorf("parse creation_time %q: %w", created, err)
	}
	return device.Restore(uid, name, brand, device.State(state), ct), nil
}
