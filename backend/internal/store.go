package internal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type FileStore struct {
	baseDir   string
	statePath string

	mu    sync.Mutex
	state *State
}

func NewFileStore(baseDir string) (*FileStore, error) {
	fs := &FileStore{
		baseDir:   baseDir,
		statePath: filepath.Join(baseDir, "state.json"),
		state:     &State{Entries: map[string]*StateEntry{}},
	}
	if err := fs.loadState(); err != nil {
		// Se non esiste, va bene; altrimenti errore
		var pathError *os.PathError
		if !errors.As(err, &pathError) {
			return nil, err
		}
	}
	return fs, nil
}

func storeKey(appID, country string) string { return fmt.Sprintf("%s-%s", appID, country) }

func (s *FileStore) ReviewsFilePath(appID, country string) string {
	return filepath.Join(s.baseDir, "reviews", fmt.Sprintf("%s-%s.jsonl", appID, country))
}

func (s *FileStore) loadState() error {
	f, err := os.Open(s.statePath)
	if err != nil {
		return err
	}
	defer f.Close()
	var st State
	if err := json.NewDecoder(f).Decode(&st); err != nil {
		return err
	}
	if st.Entries == nil {
		st.Entries = map[string]*StateEntry{}
	}
	s.state = &st
	return nil
}

func (s *FileStore) SaveState() error {
	tmp := s.statePath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.state); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.statePath)
}

// Thread-safe accessors

func (s *FileStore) GetSeenSet(appID, country string) map[string]struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ent := s.state.Entries[storeKey(appID, country)]
	set := map[string]struct{}{}
	if ent != nil {
		for _, id := range ent.SeenIDs {
			set[id] = struct{}{}
		}
	}
	return set
}

func (s *FileStore) AppendReviews(appID, country string, reviews []Review, newIDs []string) error {
	// Append JSONL
	path := s.ReviewsFilePath(appID, country)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, r := range reviews {
		b, _ := json.Marshal(r)
		if _, err := w.Write(append(b, '\n')); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Update state
	s.mu.Lock()
	defer s.mu.Unlock()
	k := storeKey(appID, country)
	ent := s.state.Entries[k]
	if ent == nil {
		ent = &StateEntry{}
	}
	ent.LastPoll = time.Now().UTC()
	// merge seenIds (append-only)
	seen := map[string]struct{}{}
	for _, id := range ent.SeenIDs {
		seen[id] = struct{}{}
	}
	for _, id := range newIDs {
		if _, ok := seen[id]; !ok {
			ent.SeenIDs = append(ent.SeenIDs, id)
			seen[id] = struct{}{}
		}
	}
	s.state.Entries[k] = ent
	return s.SaveState()
}

func (s *FileStore) ReadRecent(appID, country string, horizon time.Duration) ([]Review, error) {
	path := s.ReviewsFilePath(appID, country)
	f, err := os.Open(path)
	if err != nil {
		// file could not be there the first time we run the script
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			return []Review{}, nil
		}
		return nil, err
	}
	defer f.Close()

	cutoff := time.Now().UTC().Add(-horizon)
	out := []Review{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r Review
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			continue // skip corrupted rows
		}
		if r.SubmittedAt.After(cutoff) || r.SubmittedAt.Equal(cutoff) {
			out = append(out, r)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	// Sort (newest first)
	sort.Slice(out, func(i, j int) bool {
		return out[i].SubmittedAt.After(out[j].SubmittedAt)
	})

	return out, nil
}

func (s *FileStore) LastPoll(appID, country string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ent := s.state.Entries[storeKey(appID, country)]
	if ent == nil || ent.LastPoll.IsZero() {
		return time.Time{}, false
	}
	return ent.LastPoll, true
}
