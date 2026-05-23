// Package db provides PostgreSQL access for the Ataljanseva nagarsevak table.
//
// Table schema (derived from the exported CSV):
//
//	id              uuid PRIMARY KEY
//	full_name       text
//	name_hindi      text
//	party           text
//	pincode         integer
//	state           text
//	district        text
//	ward            text        -- ward code, e.g. "17C"
//	ward_hindi      text
//	state_hindi     text
//	district_hindi  text
//	pincode_hindi   text
//	address         text        -- municipality label, e.g. "Maharashtra-Mira Bhayander"
//	slug            text
//	is_active       boolean
//	-- many more columns exist in the full table; only the above are queried here
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // postgres driver
)

// ─────────────────────────────────────────────
// Domain types
// ─────────────────────────────────────────────

// LocationInfo holds the resolved state/district for a PIN code.
type LocationInfo struct {
	State    string
	District string
	// Hindi variants (shown when language == "mr" or "hi")
	StateHindi    string
	DistrictHindi string
}

// Ward represents a single ward entry.
type Ward struct {
	Code      string // "17C"
	CodeHindi string // "१७सी"
}

// Nagarsevak is a single representative row from the DB.
type Nagarsevak struct {
	ID        string
	FullName  string
	NameHindi string
	Party     string
	Ward      string
	Slug      string
}

// ─────────────────────────────────────────────
// Repository
// ─────────────────────────────────────────────

// Repo wraps a *sql.DB and exposes query methods used by the bot.
type Repo struct {
	db *sql.DB
}

// New opens a PostgreSQL connection using the given DSN and pings it.
//
//	dsn = "host=localhost port=5432 dbname=ataljanseva user=postgres password=secret sslmode=disable"
func New(dsn string) (*Repo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	return &Repo{db: db}, nil
}

// Close closes the underlying connection pool.
func (r *Repo) Close() error { return r.db.Close() }

// ─────────────────────────────────────────────
// PIN → location
// ─────────────────────────────────────────────

// LocationByPincode returns the state/district for a given PIN code.
// Returns sql.ErrNoRows if the PIN is unknown.
func (r *Repo) LocationByPincode(pincode string) (*LocationInfo, error) {
	row := r.db.QueryRow(`
		SELECT state, district, state_hindi, district_hindi
		FROM   political_users
		WHERE  pincode::text = $1
		  AND  is_active = true
		LIMIT  1
	`, pincode)

	var loc LocationInfo
	err := row.Scan(&loc.State, &loc.District, &loc.StateHindi, &loc.DistrictHindi)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// ─────────────────────────────────────────────
// PIN → wards
// ─────────────────────────────────────────────

// WardsByPincode returns the distinct wards for a PIN code, ordered by ward code.
func (r *Repo) WardsByPincode(pincode string) ([]Ward, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT ward, ward_hindi
		FROM   political_users
		WHERE  pincode::text = $1
		  AND  ward IS NOT NULL
		  AND  ward <> ''
		  AND  is_active = true
		ORDER  BY ward
	`, pincode)
	if err != nil {
		return nil, fmt.Errorf("WardsByPincode query: %w", err)
	}
	defer rows.Close()

	var wards []Ward
	for rows.Next() {
		var w Ward
		var hindi sql.NullString
		if err := rows.Scan(&w.Code, &hindi); err != nil {
			return nil, err
		}
		if hindi.Valid {
			w.CodeHindi = hindi.String
		}
		wards = append(wards, w)
	}
	return wards, rows.Err()
}

// ─────────────────────────────────────────────
// Ward → nagarsevaks
// ─────────────────────────────────────────────

// NagarsevaksByWard returns all active nagarsevaks for a given pincode + ward.
func (r *Repo) NagarsevaksByWard(pincode, ward string) ([]Nagarsevak, error) {
	rows, err := r.db.Query(`
		SELECT id, full_name, COALESCE(name_hindi,''), COALESCE(party,''), ward, slug
		FROM   political_users
		WHERE  pincode::text = $1
		  AND  ward = $2
		  AND  is_active = true
		ORDER  BY full_name
	`, pincode, ward)
	if err != nil {
		return nil, fmt.Errorf("NagarsevaksByWard query: %w", err)
	}
	defer rows.Close()

	var list []Nagarsevak
	for rows.Next() {
		var n Nagarsevak
		if err := rows.Scan(&n.ID, &n.FullName, &n.NameHindi, &n.Party, &n.Ward, &n.Slug); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

// ─────────────────────────────────────────────
// Lookup by ID (used after selection)
// ─────────────────────────────────────────────

// NagarsevakByID fetches a single nagarsevak by UUID.
func (r *Repo) NagarsevakByID(id string) (*Nagarsevak, error) {
	row := r.db.QueryRow(`
		SELECT id, full_name, COALESCE(name_hindi,''), COALESCE(party,''), ward, slug
		FROM   political_users
		WHERE  id = $1
		LIMIT  1
	`, id)

	var n Nagarsevak
	if err := row.Scan(&n.ID, &n.FullName, &n.NameHindi, &n.Party, &n.Ward, &n.Slug); err != nil {
		return nil, err
	}
	return &n, nil
}
