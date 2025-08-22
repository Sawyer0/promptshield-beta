package batch

import (
	"context"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/shared/errors"
)

// Item represents an item in a batch
type Item interface{}

// ProcessFunc processes a batch of items
type ProcessFunc func(ctx context.Context, items []Item) error

// Processor manages batching and processing of items
type Processor struct {
	processFn  ProcessFunc
	maxSize    int
	maxWait    time.Duration
	maxRetries int

	items     []Item
	itemsChan chan Item
	flushChan chan chan error
	closeChan chan struct{}
	closed    bool

	mu      sync.RWMutex
	wg      sync.WaitGroup
	metrics *Metrics
}

// Metrics tracks batch processor metrics
type Metrics struct {
	mu               sync.RWMutex
	ItemsProcessed   int64
	BatchesProcessed int64
	BatchesFailed    int64
	AverageBatchSize float64
	AverageWaitTime  time.Duration
	LastFlushTime    time.Time
}

// Config configures a batch processor
type Config struct {
	MaxSize    int
	MaxWait    time.Duration
	MaxRetries int
	ProcessFn  ProcessFunc
}

// DefaultConfig returns default batch processor configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSize:    100,
		MaxWait:    1 * time.Second,
		MaxRetries: 3,
	}
}

// New creates a new batch processor
func New(config *Config) *Processor {
	if config == nil {
		config = DefaultConfig()
	}

	if config.MaxSize <= 0 {
		config.MaxSize = 100
	}
	if config.MaxWait <= 0 {
		config.MaxWait = 1 * time.Second
	}

	p := &Processor{
		processFn:  config.ProcessFn,
		maxSize:    config.MaxSize,
		maxWait:    config.MaxWait,
		maxRetries: config.MaxRetries,
		items:      make([]Item, 0, config.MaxSize),
		itemsChan:  make(chan Item, config.MaxSize),
		flushChan:  make(chan chan error),
		closeChan:  make(chan struct{}),
		metrics:    &Metrics{},
	}

	p.wg.Add(1)
	go p.run()

	return p
}

// Add adds an item to the batch
func (p *Processor) Add(ctx context.Context, item Item) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.ErrProcessorClosed
	}
	p.mu.RUnlock()

	select {
	case p.itemsChan <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.ErrBatchFull
	}
}

// AddBatch adds multiple items to the batch
func (p *Processor) AddBatch(ctx context.Context, items []Item) error {
	for _, item := range items {
		if err := p.Add(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// Flush manually triggers batch processing
func (p *Processor) Flush(ctx context.Context) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.ErrProcessorClosed
	}
	p.mu.RUnlock()

	errCh := make(chan error, 1)

	select {
	case p.flushChan <- errCh:
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the batch processor
func (p *Processor) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.closeChan)
	p.wg.Wait()

	// Process any remaining items
	if len(p.items) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return p.processBatch(ctx)
	}

	return nil
}

// run is the main processing loop
func (p *Processor) run() {
	defer p.wg.Done()

	timer := time.NewTimer(p.maxWait)
	defer timer.Stop()

	for {
		select {
		case item := <-p.itemsChan:
			p.items = append(p.items, item)

			// Check if batch is full
			if len(p.items) >= p.maxSize {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				p.processBatch(ctx)
				cancel()
				p.items = p.items[:0]
				timer.Reset(p.maxWait)
			}

		case <-timer.C:
			// Process batch on timeout
			if len(p.items) > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				p.processBatch(ctx)
				cancel()
				p.items = p.items[:0]
			}
			timer.Reset(p.maxWait)

		case errCh := <-p.flushChan:
			// Manual flush requested
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := p.processBatch(ctx)
			cancel()
			p.items = p.items[:0]
			errCh <- err
			timer.Reset(p.maxWait)

		case <-p.closeChan:
			return
		}
	}
}

// processBatch processes the current batch
func (p *Processor) processBatch(ctx context.Context) error {
	if len(p.items) == 0 {
		return nil
	}

	// Copy items for processing
	batch := make([]Item, len(p.items))
	copy(batch, p.items)

	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.LastFlushTime = time.Now()
	batchSize := float64(len(batch))
	if p.metrics.BatchesProcessed > 0 {
		p.metrics.AverageBatchSize = (p.metrics.AverageBatchSize*float64(p.metrics.BatchesProcessed) + batchSize) / float64(p.metrics.BatchesProcessed+1)
	} else {
		p.metrics.AverageBatchSize = batchSize
	}
	p.metrics.mu.Unlock()

	// Process with retries
	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = p.processFn(ctx, batch)
		if lastErr == nil {
			// Update success metrics
			p.metrics.mu.Lock()
			p.metrics.ItemsProcessed += int64(len(batch))
			p.metrics.BatchesProcessed++
			p.metrics.mu.Unlock()
			return nil
		}
	}

	// Update failure metrics
	p.metrics.mu.Lock()
	p.metrics.BatchesFailed++
	p.metrics.mu.Unlock()

	return lastErr
}

// GetMetrics returns current metrics
func (p *Processor) GetMetrics() *Metrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return p.metrics
}

// Size returns the current batch size
func (p *Processor) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.items)
}

// IsFull returns true if the batch is full
func (p *Processor) IsFull() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.items) >= p.maxSize
}

// IsEmpty returns true if the batch is empty
func (p *Processor) IsEmpty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.items) == 0
}
