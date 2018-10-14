// Package rabin provides functions to calculate Rabin Fingerprints, using
// the Adler32 prime by default
//
// The algorithm used is:
//
// 	hash = (hash * PRIME + in - out * POW) % MOD
//
// Where POW is (PRIME ** windowsize) % MOD
//
// However, the modulus operations can be turned into
// a bit mask if MOD is a power of two. So this implementation takes a shift instead
// of an integer.
package rabin

import (
	"io"
	"os"
)

// New returns a Chunker configured to chunk the supplied file.
func New(f *os.File) (*Chunker, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() < MinSz {
		return &Chunker{
			single: true,
			rd:     f,
		}, nil
	}
	return &Chunker{
		buf: make([]byte, bufSz),
		rd:  f,
	}, nil
}

type Chunker struct {
	// hash state
	hash uint64
	// stream position
	pos int64
	// chunk position
	off int64

	buf []byte

	// If we only have a single chunk, we can skip messing with hashes and
	// whatnot.
	single bool

	rd *os.File
}

// FindBoundary returns the start and end boundaries of the next chunk.
func (c *Chunker) findBoundary() (int64, int64, error) {
	// We start at 0 or after a previous findBoundary call.
	start := c.pos
	c.off = 0
	c.pos += MinSz

	var ls [Window]byte        // array of bytes to slide out of the window
	var j int                  // index into the ls array
	var b byte                 // the byte we're looking at
	var n int                  // num bytes from ReadAt
	var err error              // ReadAt error
	wmask := int64(Window - 1) // turn the size of the window into a mask

	for n, err = c.rd.ReadAt(c.buf, c.pos); ; n, err = c.rd.ReadAt(c.buf, c.pos) {
		// We need to handle the bytes even if there was an error.
		for i := 0; i < n; i++ {
			b = c.buf[i]
			j = int(c.off & wmask) // avoid a %
			c.hash *= Prime
			c.hash += uint64(b)
			c.hash -= popTable[ls[j]]
			c.hash &= mod
			ls[j] = b
			c.off++
			if c.hash<<Mask == 0 || c.off > MaxSz {
				err = fpDone
				break
			}
		}
		c.pos += c.off
		switch err {
		case nil:
			continue
		case fpDone:
			return start, c.pos, nil
		default:
			return 0, 0, err
		}
	}
}

func (c *Chunker) Next() (io.ReadSeeker, error) {
	if c.single {
		_, err := c.rd.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}
		return c.rd, nil
	}

	a, b, err := c.findBoundary()
	if err != nil {
		return nil, err
	}
	return io.NewSectionReader(c.rd, a, b), nil
}

// Reset clears internal state and readies the Chunker to read from rd.
//
// If the supplied Reader is nil, the current Reader is reset.
func (c *Chunker) Reset(rd *os.File) error {
	if rd == nil {
		rd = c.rd
	}
	if _, err := rd.Seek(0, io.SeekStart); err != nil {
		return err
	}
	c.hash = 0
	c.pos = 0
	c.off = 0
	c.rd = rd
	return nil
}
