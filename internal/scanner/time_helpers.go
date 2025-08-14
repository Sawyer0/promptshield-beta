package scanner

import "time"

func timeNow() time.Time { return time.Now() }

func timeSinceMs(t time.Time) int64 { return time.Since(t).Milliseconds() }
