package pool

import (
	"context"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/shared/errors"
)

// Task represents a unit of work
type Task func(ctx context.Context) error

// Result represents the result of a task execution
type Result struct {
	ID    string
	Error error
	Data  interface{}
}

// Worker represents a worker in the pool
type Worker struct {
	id       int
	pool     *Pool
	taskChan chan *taskWrapper
	quit     chan bool
}

// taskWrapper wraps a task with its context and result channel
type taskWrapper struct {
	ctx      context.Context
	task     Task
	resultCh chan error
}

// Pool represents a worker pool
type Pool struct {
	workers    []*Worker
	taskQueue  chan *taskWrapper
	maxWorkers int
	maxQueue   int

	mu       sync.RWMutex
	closed   bool
	wg       sync.WaitGroup
	metrics  *Metrics
}

// Metrics represents pool metrics
type Metrics struct {
	mu              sync.RWMutex
	TasksSubmitted  int64
	TasksCompleted  int64
	TasksFailed     int64
	TasksInQueue    int64
	ActiveWorkers   int64
	TotalWorkers    int64
	AverageWaitTime time.Duration
	AverageExecTime time.Duration
}

// New creates a new worker pool
func New(maxWorkers, maxQueue int) *Pool {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	if maxQueue <= 0 {
		maxQueue = maxWorkers * 10
	}

	p := &Pool{
		workers:    make([]*Worker, 0, maxWorkers),
		taskQueue:  make(chan *taskWrapper, maxQueue),
		maxWorkers: maxWorkers,
		maxQueue:   maxQueue,
		metrics:    &Metrics{TotalWorkers: int64(maxWorkers)},
	}

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		w := &Worker{
			id:       i,
			pool:     p,
			taskChan: make(chan *taskWrapper),
			quit:     make(chan bool),
		}
		p.workers = append(p.workers, w)
		go w.start()
	}

	// Start dispatcher
	go p.dispatch()

	return p
}

// Submit submits a task to the pool
func (p *Pool) Submit(ctx context.Context, task Task) error {
	return p.SubmitWithTimeout(ctx, task, 0)
}

// SubmitWithTimeout submits a task with a timeout
func (p *Pool) SubmitWithTimeout(ctx context.Context, task Task, timeout time.Duration) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return errors.ErrPoolClosed
	}
	p.mu.RUnlock()

	// Create task wrapper
	wrapper := &taskWrapper{
		ctx:      ctx,
		task:     task,
		resultCh: make(chan error, 1),
	}

	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.TasksSubmitted++
	p.metrics.TasksInQueue++
	p.metrics.mu.Unlock()

	// Set timeout if specified
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		wrapper.ctx = ctx
	}

	// Submit to queue
	select {
	case p.taskQueue <- wrapper:
		// Wait for result
		select {
		case err := <-wrapper.resultCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		p.metrics.mu.Lock()
		p.metrics.TasksInQueue--
		p.metrics.mu.Unlock()
		return ctx.Err()
	default:
		p.metrics.mu.Lock()
		p.metrics.TasksInQueue--
		p.metrics.mu.Unlock()
		return errors.ErrPoolTimeout
	}
}

// SubmitAsync submits a task asynchronously
func (p *Pool) SubmitAsync(ctx context.Context, task Task) <-chan error {
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- p.Submit(ctx, task)
		close(resultCh)
	}()

	return resultCh
}

// dispatch distributes tasks to workers
func (p *Pool) dispatch() {
	for {
		select {
		case task := <-p.taskQueue:
			// Find available worker
			for _, worker := range p.workers {
				select {
				case worker.taskChan <- task:
					goto next
				default:
					continue
				}
			}
			// If no worker available, wait
			p.workers[0].taskChan <- task
		next:
		}

		p.mu.RLock()
		if p.closed && len(p.taskQueue) == 0 {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()
	}
}

// Close gracefully shuts down the pool
func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Stop accepting new tasks
	close(p.taskQueue)

	// Signal workers to stop
	for _, worker := range p.workers {
		worker.quit <- true
	}

	// Wait for workers to finish
	p.wg.Wait()

	return nil
}

// Resize changes the number of workers
func (p *Pool) Resize(newSize int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.ErrPoolClosed
	}

	if newSize <= 0 {
		newSize = 1
	}

	currentSize := len(p.workers)
	
	if newSize > currentSize {
		// Add workers
		for i := currentSize; i < newSize; i++ {
			w := &Worker{
				id:       i,
				pool:     p,
				taskChan: make(chan *taskWrapper),
				quit:     make(chan bool),
			}
			p.workers = append(p.workers, w)
			go w.start()
		}
	} else if newSize < currentSize {
		// Remove workers
		for i := newSize; i < currentSize; i++ {
			p.workers[i].quit <- true
		}
		p.workers = p.workers[:newSize]
	}

	p.metrics.mu.Lock()
	p.metrics.TotalWorkers = int64(newSize)
	p.metrics.mu.Unlock()

	return nil
}

// GetMetrics returns current pool metrics
func (p *Pool) GetMetrics() Metrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// Size returns the current number of workers
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}

// QueueSize returns the number of tasks in queue
func (p *Pool) QueueSize() int {
	return len(p.taskQueue)
}

// start starts a worker
func (w *Worker) start() {
	w.pool.wg.Add(1)
	defer w.pool.wg.Done()

	for {
		select {
		case task := <-w.taskChan:
			w.execute(task)
		case <-w.quit:
			return
		}
	}
}

// execute executes a task
func (w *Worker) execute(wrapper *taskWrapper) {
	start := time.Now()

	// Update metrics
	w.pool.metrics.mu.Lock()
	w.pool.metrics.TasksInQueue--
	w.pool.metrics.ActiveWorkers++
	w.pool.metrics.mu.Unlock()

	// Execute task
	err := wrapper.task(wrapper.ctx)

	// Update metrics
	w.pool.metrics.mu.Lock()
	w.pool.metrics.ActiveWorkers--
	w.pool.metrics.TasksCompleted++
	if err != nil {
		w.pool.metrics.TasksFailed++
	}
	execTime := time.Since(start)
	w.pool.metrics.AverageExecTime = (w.pool.metrics.AverageExecTime + execTime) / 2
	w.pool.metrics.mu.Unlock()

	// Send result
	select {
	case wrapper.resultCh <- err:
	default:
	}
}