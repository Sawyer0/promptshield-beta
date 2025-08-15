package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/audit"
)

// Append-only file-backed event store with basic in-memory index.
// Designed for replacement with a durable store (e.g., SQLite, Postgres, or Kafka) later.

type Store struct {
	mu   sync.Mutex
	file *os.File
	path string
	seq  uint64
	idx  []uint64 // byte offsets for events
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	return &Store{file: f, path: path, idx: make([]uint64, 0, 1024)}, nil
}

func (s *Store) Close() error { return s.file.Close() }

type Record struct {
	Seq   uint64      `json:"seq"`
	Event audit.Event `json:"event"`
	At    time.Time   `json:"at"`
}

func (s *Store) Append(ctx context.Context, e audit.Event) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	s.seq++
	rec := Record{Seq: s.seq, Event: e, At: time.Now().UTC()}
	pos, _ := s.file.Seek(0, os.SEEK_END)
	b, err := json.Marshal(rec)
	if err != nil {
		return 0, err
	}
	b = append(b, '\n')
	if _, err := s.file.Write(b); err != nil {
		return 0, err
	}
	// Safe conversion: pos is from Seek which returns int64, but we verify it's non-negative
	if pos < 0 {
		return 0, errors.New("invalid file position from seek")
	}
	s.idx = append(s.idx, uint64(pos))
	return s.seq, nil
}

func (s *Store) Get(ctx context.Context, seq uint64) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return Record{}, ctx.Err()
	default:
	}

	if seq == 0 || seq > s.seq {
		return Record{}, errors.New("sequence number not found")
	}

	// Use index to seek to approximate position
	if seq <= uint64(len(s.idx)) {
		offset := s.idx[seq-1]
		// Safe conversion: check for overflow before converting uint64 to int64
		if offset > math.MaxInt64 {
			return Record{}, errors.New("file offset too large for seek operation")
		}
		if _, err := s.file.Seek(int64(offset), 0); err != nil {
			return Record{}, err
		}
	} else {
		// Fallback to start of file if index is incomplete
		if _, err := s.file.Seek(0, 0); err != nil {
			return Record{}, err
		}
	}

	// Read and parse records until we find the target sequence
	decoder := json.NewDecoder(s.file)
	for {
		var rec Record
		if err := decoder.Decode(&rec); err != nil {
			return Record{}, err
		}
		if rec.Seq == seq {
			return rec, nil
		}
		if rec.Seq > seq {
			return Record{}, errors.New("sequence number not found")
		}
	}
}
