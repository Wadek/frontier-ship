package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry is one sealed evidence row. Append-only.
type Entry struct {
	Seq       int64          `json:"seq"`
	TS        string         `json:"ts"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Payload   map[string]any `json:"payload"`
	PrevHash  string         `json:"prev_hash"`
	EntryHash string         `json:"entry_hash"`
}

type Ledger struct {
	path string
	mu   sync.Mutex
}

func Open(path string) (*Ledger, error) {
	if path == "" {
		return nil, fmt.Errorf("ledger path required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	// Keep Frontier metadata out of git diffs / dirty checks.
	_ = ensureGitignore(filepath.Dir(dir), ".frontier/")
	f, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return &Ledger{path: path}, nil
}

func ensureGitignore(repoRoot, entry string) error {
	gi := filepath.Join(repoRoot, ".gitignore")
	b, err := os.ReadFile(gi)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(b), entry) {
		return nil
	}
	f, err := os.OpenFile(gi, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(b) > 0 && b[len(b)-1] != '\n' {
		_, _ = f.WriteString("\n")
	}
	_, err = f.WriteString(entry + "\n")
	return err
}

func (l *Ledger) Append(actor, action string, payload map[string]any) (*Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prev, seq, err := l.tailMeta()
	if err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	e := &Entry{
		Seq:      seq + 1,
		TS:       time.Now().UTC().Format(time.RFC3339Nano),
		Actor:    actor,
		Action:   action,
		Payload:  payload,
		PrevHash: prev,
	}
	e.EntryHash = EntryHashOf(*e)

	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return nil, err
	}
	return e, nil
}

func (l *Ledger) Tail(n int) ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	raw, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}
	lines := splitNonEmpty(string(raw))
	if n <= 0 || n > len(lines) {
		n = len(lines)
	}
	start := len(lines) - n
	out := make([]Entry, 0, n)
	for _, line := range lines[start:] {
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (l *Ledger) LastAction(action string) (*Entry, error) {
	all, err := l.Tail(500)
	if err != nil {
		return nil, err
	}
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Action == action {
			e := all[i]
			return &e, nil
		}
	}
	return nil, nil
}

// EntryHashOf returns the canonical SHA-256 evidence hash of one row.
// Append uses the same canonicalization; monitors re-verify chains with it.
func EntryHashOf(e Entry) string {
	canon, _ := json.Marshal(struct {
		Seq      int64          `json:"seq"`
		TS       string         `json:"ts"`
		Actor    string         `json:"actor"`
		Action   string         `json:"action"`
		Payload  map[string]any `json:"payload"`
		PrevHash string         `json:"prev_hash"`
	}{e.Seq, e.TS, e.Actor, e.Action, e.Payload, e.PrevHash})
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:])
}

func (l *Ledger) tailMeta() (prev string, seq int64, err error) {
	raw, err := os.ReadFile(l.path)
	if err != nil {
		return "", 0, err
	}
	lines := splitNonEmpty(string(raw))
	if len(lines) == 0 {
		return "genesis", 0, nil
	}
	var e Entry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &e); err != nil {
		return "genesis", 0, nil
	}
	return e.EntryHash, e.Seq, nil
}

func splitNonEmpty(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
