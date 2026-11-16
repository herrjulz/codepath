// Package store persists reports as JSON files with atomic replacement.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codepath/internal/reports"
)

// ErrNotFound is returned when a report id does not exist in this store.
var ErrNotFound = errors.New("report not found")

// Store is a directory of report JSON files.
type Store struct {
	Dir string
}

// Open creates the report directory if needed.
func Open(dataDir string) (*Store, error) {
	dir := filepath.Join(dataDir, "reports")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

// Save writes the report atomically.
func (s *Store) Save(rep reports.WalkthroughReport) error {
	if !reports.ValidID(rep.ID) {
		return fmt.Errorf("invalid report id")
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(filepath.Join(s.Dir, rep.ID+".json"), data)
}

// Get loads one report.
func (s *Store) Get(id string) (reports.WalkthroughReport, error) {
	if !reports.ValidID(id) {
		return reports.WalkthroughReport{}, ErrNotFound
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return reports.WalkthroughReport{}, ErrNotFound
		}
		return reports.WalkthroughReport{}, err
	}
	var rep reports.WalkthroughReport
	if err := json.Unmarshal(data, &rep); err != nil {
		return reports.WalkthroughReport{}, fmt.Errorf("read report: %w", err)
	}
	if rep.ID != id {
		return reports.WalkthroughReport{}, fmt.Errorf("report id mismatch")
	}
	return rep, nil
}

// List returns report summaries sorted newest first. urlFor builds the browser URL.
func (s *Store) List(urlFor func(id string) string) ([]reports.Summary, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	out := make([]reports.Summary, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !reports.ValidID(id) {
			continue
		}
		rep, err := s.Get(id)
		if err != nil {
			continue
		}
		out = append(out, reports.SummaryOf(rep, urlFor(rep.ID)))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt > out[j].CreatedAt
	})
	return out, nil
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".report-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	success = true
	dirf, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer dirf.Close()
	_ = dirf.Sync()
	return nil
}
