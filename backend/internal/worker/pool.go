package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"backend/internal/repository"
)

// ScoreUpdateTask represents a task to persist a score update to PostgreSQL
type ScoreUpdateTask struct {
	Username string
	Rating   int
}

// WorkerPool manages a pool of workers for asynchronous database writes
type WorkerPool struct {
	jobs         chan ScoreUpdateTask
	workerCount  int
	postgresRepo *repository.PostgresRepository
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	metrics      *PoolMetrics
}

// PoolMetrics tracks worker pool performance
type PoolMetrics struct {
	mu              sync.RWMutex
	processed       int64
	failed          int64
	backpressure    int64
	totalProcessing time.Duration
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount, queueSize int, postgresRepo *repository.PostgresRepository) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		jobs:         make(chan ScoreUpdateTask, queueSize),
		workerCount:  workerCount,
		postgresRepo: postgresRepo,
		ctx:          ctx,
		cancel:       cancel,
		metrics:      &PoolMetrics{},
	}
}

// Start initializes and starts all worker goroutines
func (wp *WorkerPool) Start() {
	log.Printf("🚀 Starting worker pool with %d workers and queue size %d", wp.workerCount, cap(wp.jobs))
	
	for i := 1; i <= wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	
	log.Printf("✓ Worker pool started successfully")
}

// worker is the main worker loop that processes jobs
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	log.Printf("Worker #%d started", id)
	
	for {
		select {
		case <-wp.ctx.Done():
			log.Printf("Worker #%d shutting down", id)
			return
			
		case task, ok := <-wp.jobs:
			if !ok {
				log.Printf("Worker #%d: Job channel closed, exiting", id)
				return
			}
			
			// Process the job with panic recovery
			wp.processTask(id, task)
		}
	}
}

// processTask handles a single score update task with error recovery
func (wp *WorkerPool) processTask(workerID int, task ScoreUpdateTask) {
	// Recover from panics to prevent worker crash
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️  Worker #%d PANIC recovered: %v (user: %s)", workerID, r, task.Username)
			wp.metrics.incrementFailed()
		}
	}()
	
	startTime := time.Now()
	
	// Create a context with timeout for the database operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Perform the database upsert
	err := wp.postgresRepo.UpsertUser(ctx, task.Username, task.Rating)
	
	processingTime := time.Since(startTime)
	
	if err != nil {
		log.Printf("❌ Worker #%d failed to persist score for %s: %v (took %v)", 
			workerID, task.Username, err, processingTime)
		wp.metrics.incrementFailed()
		return
	}
	
	log.Printf("✓ Worker #%d processed update for %s in %v", 
		workerID, task.Username, processingTime)
	
	wp.metrics.recordSuccess(processingTime)
}

// Submit attempts to add a task to the queue with backpressure handling
func (wp *WorkerPool) Submit(task ScoreUpdateTask) error {
	select {
	case wp.jobs <- task:
		// Successfully queued
		return nil
		
	default:
		// Queue is full - backpressure detected
		log.Printf("⚠️  BACKPRESSURE WARNING: Queue full, dropping Postgres write for user %s", task.Username)
		wp.metrics.incrementBackpressure()
		return fmt.Errorf("worker pool queue full (backpressure)")
	}
}

// Shutdown gracefully stops the worker pool
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	log.Printf("🛑 Shutting down worker pool...")
	
	// Close the job channel to signal no more jobs will be added
	close(wp.jobs)
	
	// Create a channel to signal when all workers are done
	done := make(chan struct{})
	
	go func() {
		wp.wg.Wait()
		close(done)
	}()
	
	// Wait for workers to finish with timeout
	select {
	case <-done:
		log.Printf("✓ All workers finished processing remaining jobs")
		wp.printMetrics()
		return nil
		
	case <-time.After(timeout):
		wp.cancel() // Force cancel remaining operations
		log.Printf("⚠️  Worker pool shutdown timed out after %v", timeout)
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// GetMetrics returns a snapshot of the pool metrics
func (wp *WorkerPool) GetMetrics() map[string]interface{} {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	
	avgProcessing := time.Duration(0)
	if wp.metrics.processed > 0 {
		avgProcessing = wp.metrics.totalProcessing / time.Duration(wp.metrics.processed)
	}
	
	return map[string]interface{}{
		"processed":            wp.metrics.processed,
		"failed":               wp.metrics.failed,
		"backpressure_events":  wp.metrics.backpressure,
		"avg_processing_time":  avgProcessing.String(),
		"queue_utilization":    fmt.Sprintf("%d/%d", len(wp.jobs), cap(wp.jobs)),
	}
}

// printMetrics logs the final metrics
func (wp *WorkerPool) printMetrics() {
	metrics := wp.GetMetrics()
	log.Printf("📊 Worker Pool Metrics:")
	log.Printf("   - Processed: %v", metrics["processed"])
	log.Printf("   - Failed: %v", metrics["failed"])
	log.Printf("   - Backpressure Events: %v", metrics["backpressure_events"])
	log.Printf("   - Avg Processing Time: %v", metrics["avg_processing_time"])
}

// Metrics helper methods
func (pm *PoolMetrics) recordSuccess(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.processed++
	pm.totalProcessing += duration
}

func (pm *PoolMetrics) incrementFailed() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.failed++
}

func (pm *PoolMetrics) incrementBackpressure() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.backpressure++
}
