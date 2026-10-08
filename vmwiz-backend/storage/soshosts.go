package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/lib/pq"
)

// GetSOSHosts reads the current list, including changes from other instances.
func (s *postgresstorage) GetSOSHosts(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT hostname FROM osscan_sos_host ORDER BY hostname`)
	if err != nil {
		return nil, fmt.Errorf("GetSOSHosts: %w", err)
	}
	defer rows.Close()

	hosts := []string{}
	for rows.Next() {
		var host string
		if err := rows.Scan(&host); err != nil {
			return nil, fmt.Errorf("GetSOSHosts: %w", err)
		}
		hosts = append(hosts, host)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetSOSHosts: %w", err)
	}
	return hosts, nil
}

// SetSOSHosts normalizes and replaces the list atomically. Concurrent saves
// are serialized so the last save replaces the whole list, even when empty.
func (s *postgresstorage) SetSOSHosts(ctx context.Context, hosts []string) ([]string, error) {
	seen := map[string]struct{}{}
	clean := []string{}
	stripWhitespace := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "")
	for _, host := range hosts {
		host = stripWhitespace.Replace(host)
		if host == "" {
			continue
		}
		if _, exists := seen[host]; !exists {
			seen[host] = struct{}{}
			clean = append(clean, host)
		}
	}
	sort.Strings(clean)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SetSOSHosts: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `LOCK TABLE osscan_sos_host IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return nil, fmt.Errorf("SetSOSHosts: lock: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM osscan_sos_host`); err != nil {
		return nil, fmt.Errorf("SetSOSHosts: delete: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO osscan_sos_host (hostname) SELECT unnest($1::text[])`, pq.Array(clean)); err != nil {
		return nil, fmt.Errorf("SetSOSHosts: insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SetSOSHosts: commit: %w", err)
	}
	return clean, nil
}
