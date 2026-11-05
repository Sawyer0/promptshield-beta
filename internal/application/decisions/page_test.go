package decisions

import (
	"sync"
	"testing"
)

func TestAddAssignsIncreasingSequenceNumbers(t *testing.T) {
	log := NewLog(3)
	for range 5 {
		log.Add(Record{Seq: 99, Decision: Allow}) // caller-supplied Seq is ignored
	}
	got := log.Recent(Query{})
	if len(got) != 3 || got[0].Seq != 5 || got[1].Seq != 4 || got[2].Seq != 3 {
		t.Fatalf("want seqs 5,4,3 after eviction, got %+v", got)
	}
	if data := got[0].EventData(); data["seq"] != int64(5) {
		t.Fatalf("event payload should carry seq, got %v", data["seq"])
	}
}

func TestPageCursorBoundaries(t *testing.T) {
	log := NewLog(10)
	for range 7 {
		log.Add(Record{Decision: Allow})
	}
	cases := []struct {
		q         Query
		first     int64
		count     int
		wantNext  int64
		situation string
	}{
		{Query{Limit: 3}, 7, 3, 5, "first page"},
		{Query{Limit: 3, Before: 5}, 4, 3, 2, "middle page"},
		{Query{Limit: 3, Before: 2}, 1, 1, 0, "last partial page"},
		{Query{Limit: 3, Before: 1}, 0, 0, 0, "cursor at the oldest record"},
		{Query{Limit: 3, Before: 1_000}, 7, 3, 5, "cursor ahead of the log"},
		{Query{Limit: 7}, 7, 7, 0, "page exactly fits"},
	}
	for _, tc := range cases {
		p := log.Page(tc.q)
		if len(p.Records) != tc.count || p.Next != tc.wantNext || (tc.count > 0 && p.Records[0].Seq != tc.first) {
			t.Errorf("%s: got %d records (first %+v) next=%d", tc.situation, len(p.Records), p.Records, p.Next)
		}
	}
}

func TestPageIsSafeUnderConcurrentAdds(t *testing.T) {
	log := NewLog(2000)
	for range 1000 {
		log.Add(Record{Decision: Allow})
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 5000 {
			log.Add(Record{Decision: Deny})
		}
	}()
	// Page through the log while writers evict and append; within one pass
	// sequence numbers must strictly decrease, so nothing repeats.
	for range 20 {
		last := int64(-1)
		q := Query{Limit: 137}
		for {
			p := log.Page(q)
			for _, r := range p.Records {
				if last != -1 && r.Seq >= last {
					t.Fatalf("seq %d after %d: duplicate or out of order", r.Seq, last)
				}
				last = r.Seq
			}
			if p.Next == 0 {
				break
			}
			q.Before = p.Next
		}
	}
	wg.Wait()
}
