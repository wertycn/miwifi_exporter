package concurrent

import (
	"context"
	"sync"
	"time"
)

// WorkerPool represents a pool of workers for concurrent tasks
type WorkerPool struct {
	workers   int
	taskChan  chan Task
	resultChan chan Result
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// Task represents a unit of work
type Task struct {
	ID   int
	Work func() (interface{}, error)
}

// Result represents the result of a task
type Result struct {
	ID     int
	Value  interface{}
	Error  error
	Elapsed time.Duration
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		workers:    workers,
		taskChan:   make(chan Task, workers*2),
		resultChan: make(chan Result, workers*2),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.taskChan)
	wp.wg.Wait()
	close(wp.resultChan)
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) {
	select {
	case wp.taskChan <- task:
	case <-wp.ctx.Done():
		// Pool is being stopped
	}
}

// Results returns a channel for receiving results
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultChan
}

// worker processes tasks from the task channel
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	for {
		select {
		case task, ok := <-wp.taskChan:
			if !ok {
				return // Channel closed, worker should exit
			}
			
			start := time.Now()
			value, err := task.Work()
			elapsed := time.Since(start)
			
			result := Result{
				ID:      task.ID,
				Value:   value,
				Error:   err,
				Elapsed: elapsed,
			}
			
			select {
			case wp.resultChan <- result:
			case <-wp.ctx.Done():
				return
			}
			
		case <-wp.ctx.Done():
			return
		}
	}
}

// ExecuteWithTimeout executes tasks with a timeout
func ExecuteWithTimeout(ctx context.Context, tasks []Task, timeout time.Duration) ([]Result, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	
	// Create worker pool with appropriate number of workers
	workers := min(len(tasks), 4) // Limit to 4 concurrent workers
	pool := NewWorkerPool(workers)
	pool.Start()
	defer pool.Stop()
	
	// Submit tasks
	for i, task := range tasks {
		task.ID = i
		pool.Submit(task)
	}
	
	// Collect results with timeout
	results := make([]Result, len(tasks))
	received := 0
	
	resultTimeout := time.After(timeout)
	
	for received < len(tasks) {
		select {
		case result := <-pool.Results():
			results[result.ID] = result
			received++
			
		case <-ctx.Done():
			return nil, ctx.Err()
			
		case <-resultTimeout:
			return nil, &TimeoutError{
				Timeout:    timeout,
				Received:   received,
				Total:      len(tasks),
				Message:    "task execution timeout",
			}
		}
	}
	
	return results, nil
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	Timeout   time.Duration
	Received  int
	Total     int
	Message   string
}

func (e *TimeoutError) Error() string {
	return e.Message
}

func (e *TimeoutError) TimeoutReached() bool {
	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}