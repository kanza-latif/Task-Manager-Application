package repository

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"

	"taskmanager/internal/domain"
)

type CGNATRepository struct {
	db *sql.DB
}

func NewCGNATRepository(db *sql.DB) *CGNATRepository {
	return &CGNATRepository{db: db}
}

// List fetches all rows from cgnat_table.
func (r *CGNATRepository) List() ([]*domain.CGNAT, error) {
	rows, err := r.db.Query(
		`SELECT id, private_ip, public_ip, start_port, end_port FROM cgnat_table ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("cgnat list: %w", err)
	}
	defer rows.Close()
	return scanCGNATRows(rows)
}

// FindByPrivateIP looks up a CGNAT entry by private IP string.
func (r *CGNATRepository) FindByPrivateIP(ip string) ([]*domain.CGNAT, error) {
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid IP address: %q", ip)
	}
	rows, err := r.db.Query(
		`SELECT id, private_ip, public_ip, start_port, end_port
		 FROM cgnat_table
		 WHERE private_ip = INET_ATON(?)`, ip,
	)
	if err != nil {
		return nil, fmt.Errorf("cgnat find by private_ip: %w", err)
	}
	defer rows.Close()
	return scanCGNATRows(rows)
}

func (r *CGNATRepository) FindByPublicIP(ip string) ([]*domain.CGNAT, error) {
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid IP address: %q", ip)
	}
	rows, err := r.db.Query(
		`SELECT id, private_ip, public_ip, start_port, end_port
		 FROM cgnat_table
		 WHERE public_ip = INET_ATON(?)
		 ORDER BY start_port`, ip,
	)
	if err != nil {
		return nil, fmt.Errorf("cgnat find by public_ip: %w", err)
	}
	defer rows.Close()
	return scanCGNATRows(rows)
}

// helpers

func scanCGNATRows(rows *sql.Rows) ([]*domain.CGNAT, error) {
	var results []*domain.CGNAT
	for rows.Next() {
		var c domain.CGNAT
		var privRaw, pubRaw []byte

		if err := rows.Scan(&c.ID, &privRaw, &pubRaw, &c.StartPort, &c.EndPort); err != nil {
			return nil, fmt.Errorf("cgnat scan: %w", err)
		}

		c.PrivateIP = rawToIP(privRaw)
		c.PublicIP = rawToIP(pubRaw)
		results = append(results, &c)
	}
	return results, rows.Err()
}

// rawToIP converts whatever MySQL returns for VARBINARY(16) back to dotted-decimal.
// MySQL can store IPs in 3 different ways:
//   - 4 bytes binary  (written via INET_ATON)         e.g. [CB 00 71 1E]
//   - 8 bytes binary  (integer stored as binary)       e.g. [00 00 00 00 CB 00 71 1E]
//   - plain string    (written as "3405803806" or "203.0.113.30")
func rawToIP(b []byte) string {
	// Case 1 — 4 byte binary (standard INET_ATON result)
	if len(b) == 4 {
		return net.IP(b).String()
	}

	// Case 2 — 8 byte binary (big-endian uint64 stored as binary)
	if len(b) == 8 {
		n := binary.BigEndian.Uint64(b)
		return intToIP(n)
	}

	// Case 3 — plain string, could be:
	//   "3405803806"    → integer representation
	//   "203.0.113.30"  → already dotted-decimal
	s := string(b)

	// try parsing as integer first
	n, err := strconv.ParseUint(s, 10, 64)
	if err == nil {
		return intToIP(n)
	}

	// already a valid dotted-decimal IP, return as-is
	if net.ParseIP(s) != nil {
		return s
	}

	// fallback — return raw string so nothing is lost
	return s
}

// intToIP converts a uint64 integer like 3405803806 → "203.0.113.30"
// using a hashmap of the 4 octets for clarity.
func intToIP(n uint64) string {
	octets := map[string]uint64{
		"first":  (n >> 24) & 0xFF,
		"second": (n >> 16) & 0xFF,
		"third":  (n >> 8) & 0xFF,
		"fourth": n & 0xFF,
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		octets["first"],
		octets["second"],
		octets["third"],
		octets["fourth"],
	)
}
