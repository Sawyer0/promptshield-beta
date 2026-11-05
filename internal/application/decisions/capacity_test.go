package decisions

import "testing"

func TestParseCapacity(t *testing.T) {
	cases := []struct {
		raw     string
		want    int
		wantErr bool
	}{
		{"", DefaultCapacity, false},
		{"   ", DefaultCapacity, false},
		{"2000", 2000, false},
		{" 750 ", 750, false},
		{"10", 10, false},
		{"50000", 50000, false},
		{"9", DefaultCapacity, true},
		{"0", DefaultCapacity, true},
		{"-5", DefaultCapacity, true},
		{"50001", DefaultCapacity, true},
		{"1e3", DefaultCapacity, true},
		{"lots", DefaultCapacity, true},
	}
	for _, tc := range cases {
		got, err := ParseCapacity(tc.raw)
		if got != tc.want || (err != nil) != tc.wantErr {
			t.Errorf("ParseCapacity(%q) = %d, %v; want %d, err=%v", tc.raw, got, err, tc.want, tc.wantErr)
		}
	}
}

func TestNewLogFromEnvSizesTheRing(t *testing.T) {
	t.Setenv(CapacityEnv, "12")
	l := NewLogFromEnv()
	for i := 0; i < 20; i++ {
		l.Add(Record{RequestID: string(rune('a' + i)), Decision: Allow})
	}
	s := l.Stats()
	if s.Capacity != 12 || s.Retained != 12 || s.Total != 20 || len(l.Recent(Query{})) != 12 {
		t.Fatalf("capacity=%d retained=%d total=%d recent=%d", s.Capacity, s.Retained, s.Total, len(l.Recent(Query{})))
	}

	t.Setenv(CapacityEnv, "999999")
	if got := NewLogFromEnv().Stats().Capacity; got != DefaultCapacity {
		t.Fatalf("out-of-range value gave capacity %d, want %d", got, DefaultCapacity)
	}
}
