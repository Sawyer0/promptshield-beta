package scanner

import (
	"context"
	"runtime"
	"sort"
	"sync"

	"github.com/promptshield/promptshield/pkg/types"
	"github.com/sourcegraph/conc/pool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ScanPathsOrdered scans the provided paths using a worker pool and returns
// results in deterministic order (sorted by path). The number of workers is
// auto-scaled when maxWorkers <= 0. The pendingWindow caps the number of
// in-flight jobs ahead of the next result to keep memory bounded; when <= 0 a
// sane default is used.
func (s *Scanner) ScanPathsOrdered(ctx context.Context, paths []string, maxWorkers int, pendingWindow int, gatherAllErrors bool) ([]types.ScanResult, error) {
	// Deterministic order by path
	sort.Strings(paths)
	total := len(paths)
	if total == 0 {
		return nil, nil
	}
	// Context used by pool for cancellation/propagation
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers > total {
		maxWorkers = total
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	if pendingWindow <= 0 {
		pendingWindow = 256
	}

	// Pool with panic recovery and chosen error behavior
	results := make([]types.ScanResult, total)
	if gatherAllErrors {
		ep := pool.New().WithErrors().WithMaxGoroutines(maxWorkers).WithContext(ctx)
		for i := 0; i < total; i++ {
			idx := i
			path := paths[i]
			ep.Go(func(ctx context.Context) error {
				ctxSpan, span := s.tracer.Start(ctx, "scan_file", trace.WithAttributes(attribute.String("path", path)))
				start := timeNow()
				r, err := s.ScanFile(ctxSpan, path)
				if err == nil {
					r.DurationMs = timeSinceMs(start)
				}
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}
				span.End()
				results[idx] = r
				return err
			})
		}
		if err := ep.Wait(); err != nil {
			return nil, err
		}
		return results, nil
	}

	p := pool.New().WithErrors().WithMaxGoroutines(maxWorkers).WithContext(ctx)
	for i := 0; i < total; i++ {
		idx := i
		path := paths[i]
		p.Go(func(ctx context.Context) error {
			ctxSpan, span := s.tracer.Start(ctx, "scan_file", trace.WithAttributes(attribute.String("path", path)))
			start := timeNow()
			r, err := s.ScanFile(ctxSpan, path)
			if err == nil {
				r.DurationMs = timeSinceMs(start)
			}
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			span.End()
			results[idx] = r
			return err
		})
	}
	if err := p.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// ScanPathsOrderedStream scans paths with a worker pool and emits results in
// deterministic order via the emit callback as soon as they become available.
// Memory is bounded by pendingWindow. If emit returns an error, scanning is
// cancelled and the error is returned after draining workers.
func (s *Scanner) ScanPathsOrderedStream(
	ctx context.Context,
	paths []string,
	maxWorkers int,
	pendingWindow int,
	gatherAllErrors bool,
	emit func(types.ScanResult) error,
) error {
	// Deterministic order by path
	sort.Strings(paths)
	total := len(paths)
	if total == 0 {
		return nil
	}
	// Derive a cancellable context to stop workers cleanly on first error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers > total {
		maxWorkers = total
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	if pendingWindow <= 0 {
		pendingWindow = 256
	}

	type item struct {
		idx  int
		path string
	}
	type out struct {
		idx int
		res types.ScanResult
		err error
	}

	jobs := make(chan item)
	outs := make(chan out, pendingWindow)
	wg := &sync.WaitGroup{}

	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range jobs {
				ctxSpan, span := s.tracer.Start(ctx, "scan_file", trace.WithAttributes(attribute.String("path", it.path)))
				start := timeNow()
				r, err := s.ScanFile(ctxSpan, it.path)
				if err == nil {
					r.DurationMs = timeSinceMs(start)
				}
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}
				// Try to send result; respect cancellation
				select {
				case outs <- out{idx: it.idx, res: r, err: err}:
				case <-ctx.Done():
					span.End()
					return
				}
				span.End()
			}
		}()
	}

	// Feed jobs
	go func() {
		for i := 0; i < total; i++ {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- item{idx: i, path: paths[i]}:
			}
		}
		close(jobs)
	}()

	// Close outs when workers finish
	go func() { wg.Wait(); close(outs) }()

	pendingMap := make(map[int]types.ScanResult)
	next := 0
	var firstErr error
	for o := range outs {
		if o.err != nil && firstErr == nil {
			firstErr = o.err
		}
		pendingMap[o.idx] = o.res
		for {
			r, ok := pendingMap[next]
			if !ok {
				break
			}
			if err := emit(r); err != nil && firstErr == nil {
				firstErr = err
			}
			delete(pendingMap, next)
			next++
		}
	}
	// All workers done; outs is closed above

	// Final flush of remaining contiguous results
	for {
		r, ok := pendingMap[next]
		if !ok {
			break
		}
		if err := emit(r); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(pendingMap, next)
		next++
	}
	if firstErr != nil {
		return firstErr
	}
	return nil
}
