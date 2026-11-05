package decisions

// Page is one slice of a paged listing, newest first.
type Page struct {
	Records []Record
	// Next is the Before value that continues the listing, or 0 when no
	// older matching record is retained.
	Next int64
}

// Page returns up to q.Limit matching records older than q.Before (newest
// first when Before is 0) and the cursor for the next page. Sequence numbers
// never change, so records added between calls do not shift later pages;
// records evicted between calls are simply no longer returned.
func (l *Log) Page(q Query) Page {
	if q.Limit <= 0 {
		return Page{Records: l.Recent(q)}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	records := l.scan(q, q.Limit+1)
	if len(records) <= q.Limit {
		return Page{Records: records}
	}
	records = records[:q.Limit]
	return Page{Records: records, Next: records[len(records)-1].Seq}
}
