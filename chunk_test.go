package rabin_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"

	"github.com/hdonnay/rabin"
)

const seed = 4 // IEEE standard random number

func genFile(name string, seed int64) (*os.File, error) {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		f, err := os.Create(name)
		if err != nil {
			return nil, err
		}
		r := io.LimitReader(rand.New(rand.NewSource(seed)), 100<<20) // ~100MB
		if _, err := io.Copy(f, r); err != nil {
			return nil, err
		}
		f.Close()
	}
	return os.Open(name)
}

type pair struct {
	name string
	seed int64
}

type chunkStat struct {
	name       string
	start, end int64
	checksum   []byte
}

var (
	files = []pair{
		{
			name: fmt.Sprintf("testdata/seed-%d", 4),
			seed: 4,
		},
		{
			name: fmt.Sprintf("testdata/seed-%d", 16),
			seed: 16,
		},
	}
)

// This tests that the chunking is deterministic.
func TestRepeatChunk(t *testing.T) {
	for _, p := range files {
		t.Run(fmt.Sprintf("seed(%d)", p.seed), repeatchunk(p))
	}
}

func repeatchunk(p pair) func(t *testing.T) {
	var want [][]byte
	return func(t *testing.T) {
		t.Parallel()
		f, err := genFile(p.name, p.seed)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		h := sha256.New()

		// Generate a list of checksums
		c, err := rabin.New(f)
		if err != nil {
			t.Fatal(err)
		}

		s, err := c.Next()
		for ; err == nil; s, err = c.Next() {
			if _, err := io.Copy(h, s); err != nil {
				t.Fatal(err)
			}
			want = append(want, h.Sum(nil))
			t.Logf("%x", h.Sum(nil))
			h.Reset()
		}

		s, err = c.Next()
		i := 0
		for ; err == nil; s, err = c.Next() {
			if _, err := io.Copy(h, s); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(want[i], h.Sum(nil)) {
				t.Fatalf("exp: %x, got: %x", want[i], h.Sum(nil))
			}
			h.Reset()
			i++
		}
	}
}

func TestReassembleChunk(t *testing.T) {
	for _, p := range files {
		t.Run(fmt.Sprintf("seed(%d)", p.seed), reassemblechunk(p))
	}
}

func reassemblechunk(p pair) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		f, err := genFile(p.name, p.seed)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		h := sha256.New()

		// find known-good checksum
		if _, err := io.Copy(h, f); err != nil {
			t.Fatal(err)
		}
		exp := h.Sum(nil)
		h.Reset()

		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		c, err := rabin.New(f)
		if err != nil {
			t.Fatal(err)
		}

		for s, err := c.Next(); err == nil; s, err = c.Next() {
			if _, err := io.Copy(h, s); err != nil {
				t.Fatal(err)
			}
		}

		if got := h.Sum(nil); !bytes.Equal(exp, got) {
			t.Fatalf("exp: %x, got: %x", exp, got)
		}
	}
}
