package scanner

import "testing"

func TestBloom_Basic(t *testing.T) {
	b := NewBloom(1000, 0.02)
	terms := [][]byte{[]byte("secret"), []byte("password"), []byte("token1234")}
	for _, x := range terms {
		b.Add(x)
	}
	for _, x := range terms {
		if !b.MightContain(x) {
			t.Fatalf("false negative for %q", string(x))
		}
	}
	// Most random strings should be rejected
	if b.MightContain([]byte("unlikely-random-phrase-zzzz")) {
		t.Log("bloom reported maybe for unlikely term (expected occasional FP)")
	}
}
