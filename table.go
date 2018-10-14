package rabin

import (
	"fmt"
	"math/big"
)

const (
	// This is the Alder32 prime
	Prime = 65521
	// Window size
	Window = 16
	// Making the modulus a power of 2 allows us to do an and instead of a
	// mod inside the tight loop.
	Shift = 31
	Mask  = 64 - 13

	mod   = (1 << Shift) - 1
	bufSz = 8192

	// Min/Max is about 4MiB and 5MiB, respectively.
	MinSz = 4 << 20
	MaxSz = 5 << 20
)

var (
	// All chunkers share this table.
	// If you modify this outside of init(), you deserve what happens.
	//
	// The alternative is to pregenerate this and make a const array.
	popTable [1 << 8]uint64

	fpDone = fmt.Errorf("we found it: return a start/stop")
)

// Generate the shift table on package load.
func init() {
	bigPrime := big.NewInt(int64(Prime))
	bigWindow := big.NewInt(int64(Window))
	bigMod := big.NewInt(1 << Shift)
	var i int64
	for i = 0; i < 1<<8; i++ {
		n := big.NewInt(i)
		popTable[i] = n.Mul(n, (&big.Int{}).Exp(bigPrime, bigWindow, bigMod)).Uint64()
	}
}
