package scanner

import bbloom "github.com/bits-and-blooms/bloom/v3"

// Bloom wraps a third-party bloom filter to keep API compatibility for now.
type Bloom struct{ inner *bbloom.BloomFilter }

// NewBloom creates a Bloom filter sized for n items with expected false positive rate p.
func NewBloom(n int, p float64) *Bloom {
    if n <= 0 { n = 1 }
    if p <= 0 || p >= 1 { p = 0.02 }
    // Use bits-and-blooms allocator (m and k chosen internally when using NewWithEstimates)
    bf := bbloom.NewWithEstimates(uint(n), p) //nolint:gosec // n is validated to be positive
    return &Bloom{inner: bf}
}

// Add inserts a term into the Bloom filter.
func (b *Bloom) Add(term []byte) { if b != nil && len(term) > 0 { b.inner.Add(term) } }

// MightContain returns true if the term may be in the set.
func (b *Bloom) MightContain(term []byte) bool { if b == nil || len(term) == 0 { return true }; return b.inner.Test(term) }
