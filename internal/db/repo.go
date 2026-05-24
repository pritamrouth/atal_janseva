// Package db provides PostgreSQL access for the Ataljanseva nagarsevak table.
// Connection pool is tuned for high concurrency via config values.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// ─────────────────────────────────────────────
// Domain types
// ─────────────────────────────────────────────

type LocationInfo struct {
	State         string
	District      string
	StateHindi    string
	DistrictHindi string
}

type Ward struct {
	Code      string
	CodeHindi string
}

// Nagarsevak holds a single representative's details.
// ProfilePhoto is a public URL stored in the profile_photo column.
type Nagarsevak struct {
	ID           string
	FullName     string
	NameHindi    string
	Party        string
	Ward         string
	Slug         string
	ProfilePhoto string // TASK 2 – public URL for the representative's photo
}

// ─────────────────────────────────────────────
// Repository
// ─────────────────────────────────────────────

type Repo struct {
	db *sql.DB
}

// New opens a PostgreSQL connection pool with tuned settings.
func New(dsn string, maxOpen, maxIdle int) (*Repo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	return &Repo{db: db}, nil
}

func (r *Repo) Close() error { return r.db.Close() }

// ─────────────────────────────────────────────
// Queries  (all accept context for cancellation)
// ─────────────────────────────────────────────

func (r *Repo) LocationByPincode(ctx context.Context, pincode string) (*LocationInfo, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT state, district,
		       COALESCE(state_hindi,''),
		       COALESCE(district_hindi,'')
		FROM   political_users
		WHERE  pincode::text = $1
		  AND  is_active = true
		LIMIT  1
	`, pincode)

	var loc LocationInfo
	if err := row.Scan(&loc.State, &loc.District, &loc.StateHindi, &loc.DistrictHindi); err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *Repo) WardsByPincode(ctx context.Context, pincode string) ([]Ward, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ward, COALESCE(ward_hindi,'')
		FROM   political_users
		WHERE  pincode::text = $1
		  AND  ward IS NOT NULL
		  AND  ward <> ''
		  AND  is_active = true
		ORDER  BY ward
	`, pincode)
	if err != nil {
		return nil, fmt.Errorf("WardsByPincode: %w", err)
	}
	defer rows.Close()

	var wards []Ward
	for rows.Next() {
		var w Ward
		if err := rows.Scan(&w.Code, &w.CodeHindi); err != nil {
			return nil, err
		}
		wards = append(wards, w)
	}
	return wards, rows.Err()
}

// NagarsevaksByWard returns all active representatives for a pincode+ward.
// It reads the profile_photo column (TASK 2).
func (r *Repo) NagarsevaksByWard(ctx context.Context, pincode, ward string) ([]Nagarsevak, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,
		       full_name,
		       COALESCE(name_hindi,''),
		       COALESCE(party,''),
		       ward,
		       slug,
		       COALESCE(profile_photo,'')
		FROM   political_users
		WHERE  pincode::text = $1
		  AND  ward = $2
		  AND  is_active = true
		ORDER  BY full_name
	`, pincode, ward)
	if err != nil {
		return nil, fmt.Errorf("NagarsevaksByWard: %w", err)
	}
	defer rows.Close()

	var list []Nagarsevak
	for rows.Next() {
		var n Nagarsevak
		if err := rows.Scan(
			&n.ID, &n.FullName, &n.NameHindi,
			&n.Party, &n.Ward, &n.Slug,
			&n.ProfilePhoto, // TASK 2
		); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

// NagarsevakByID looks up a single representative by primary key.
// It reads the profile_photo column (TASK 2).
func (r *Repo) NagarsevakByID(ctx context.Context, id string) (*Nagarsevak, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id,
		       full_name,
		       COALESCE(name_hindi,''),
		       COALESCE(party,''),
		       ward,
		       slug,
		       COALESCE(profile_photo,'')
		FROM   political_users
		WHERE  id = $1
		LIMIT  1
	`, id)

	var n Nagarsevak
	if err := row.Scan(
		&n.ID, &n.FullName, &n.NameHindi,
		&n.Party, &n.Ward, &n.Slug,
		&n.ProfilePhoto, // TASK 2
	); err != nil {
		return nil, err
	}
	return &n, nil
}