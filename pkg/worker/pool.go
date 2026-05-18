package worker

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// JobType represents the type of background job to process
type JobType string

const (
	// New job types requested by user
	JobVideoTranscode   JobType = "video_transcode"
	JobImageScan        JobType = "image_scan"
	JobEmailSend        JobType = "email_send"
	JobNotificationPush JobType = "notification_push"
	JobMediaDelete      JobType = "media_delete"

	// Existing job types for backward compatibility
	JobVideoTranscoding  JobType = "video_transcoding"
	JobStreamArchive     JobType = "stream_archive"
	JobDailyReport      JobType = "daily_report"
	JobNotificationBatch JobType = "notification_batch"
	JobXPBatchUpdate     JobType = "xp_batch_update"
)

// Job represents a background task that needs to be processed
type Job struct {
	ID         string          `json:"id"`
	Type       JobType         `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Priority   int             `json:"priority"`     // Kept for backward compatibility
	Retry      int             `json:"retry"`        // Kept for backward compatibility (tracks current attempt count)
	MaxRetry   int             `json:"max_retry"`    // Kept for backward compatibility (tracks max attempts allowed)
	RetryCount int             `json:"retry_count"`  // Internal track
	MaxRetries int             `json:"max_retries"`  // Internal track
	CreatedAt  time.Time       `json:"created_at"`
}

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, job *Job) error

// WorkerPool manages a channel-based queue of jobs and a set of concurrent workers
type WorkerPool struct {
	workerCount int
	jobQueue    chan *Job
	handlers    map[JobType]JobHandler
	logger      *zap.Logger
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isClosed    bool
}

// NewWorkerPool creates and returns a new WorkerPool
func NewWorkerPool(workerCount int, queueSize int, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workerCount: workerCount,
		jobQueue:    make(chan *Job, queueSize),
		handlers:    make(map[JobType]JobHandler),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// NewPool creates a new WorkerPool (backward compatibility wrapper)
func NewPool(workerCount int, logger *zap.Logger) *WorkerPool {
	return NewWorkerPool(workerCount, 100, logger)
}

// RegisterHandler registers a new handler for a specific job type
func (wp *WorkerPool) RegisterHandler(jobType JobType, handler JobHandler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.handlers[jobType] = handler
}

// Submit enqueues a job for processing. Returns error if pool is closed.
func (wp *WorkerPool) Submit(job *Job) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	if wp.isClosed {
		return errors.New("worker pool is closed")
	}

	// Align backward compatible fields with new ones
	if job.MaxRetries == 0 && job.MaxRetry > 0 {
		job.MaxRetries = job.MaxRetry
	}
	if job.RetryCount == 0 && job.Retry > 0 {
		job.RetryCount = job.Retry
	}

	select {
	case wp.jobQueue <- job:
		wp.logger.Debug("Job submitted to queue", zap.String("job_id", job.ID), zap.String("type", string(job.Type)))
		return nil
	default:
		wp.logger.Warn("Job queue is full, discarding job", zap.String("job_id", job.ID))
		return errors.New("job queue is full")
	}
}

// Enqueue submits a job to the queue (backward compatibility wrapper)
func (wp *WorkerPool) Enqueue(job *Job) {
	_ = wp.Submit(job)
}

// Start launches the workers in background goroutines
func (wp *WorkerPool) Start() {
	wp.logger.Info("Starting worker pool", zap.Int("workers", wp.workerCount))
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop gracefully shuts down the pool (backward compatibility wrapper)
func (wp *WorkerPool) Stop() {
	_ = wp.Shutdown(30 * time.Second)
}

// Shutdown gracefully shuts down the worker pool, waiting for running jobs to finish
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	wp.mu.Lock()
	if wp.isClosed {
		wp.mu.Unlock()
		return nil
	}
	wp.isClosed = true
	wp.mu.Unlock()

	wp.logger.Info("Initiating graceful shutdown of worker pool")
	
	// Close the job queue channel so no more jobs can be submitted,
	// and workers will terminate once they finish reading existing jobs in the channel.
	close(wp.jobQueue)

	// Wait for all workers to finish processing
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.cancel()
		wp.logger.Info("Worker pool shut down gracefully")
		return nil
	case <-time.After(timeout):
		wp.cancel()
		wp.logger.Warn("Worker pool shutdown timed out, forcing exit")
		return errors.New("worker pool shutdown timed out")
	}
}

func (wp *WorkerPool) worker(workerID int) {
	defer wp.wg.Done()
	wp.logger.Debug("Worker started", zap.Int("worker_id", workerID))

	for {
		select {
		case <-wp.ctx.Done():
			wp.logger.Debug("Worker received cancellation, exiting", zap.Int("worker_id", workerID))
			return
		case job, ok := <-wp.jobQueue:
			if !ok {
				wp.logger.Debug("Worker queue closed, exiting", zap.Int("worker_id", workerID))
				return
			}
			wp.process(job)
		}
	}
}

func (wp *WorkerPool) process(job *Job) {
	wp.logger.Info("Processing job", zap.String("job_id", job.ID), zap.String("type", string(job.Type)))

	wp.mu.RLock()
	handler, exists := wp.handlers[job.Type]
	wp.mu.RUnlock()

	if !exists {
		wp.logger.Error("No handler registered for job type", zap.String("job_id", job.ID), zap.String("type", string(job.Type)))
		return
	}

	// Gracefully handle panics in job handler execution
	defer func() {
		if r := recover(); r != nil {
			wp.logger.Error("Job handler panicked", zap.String("job_id", job.ID), zap.Any("panic", r))
			wp.handleFailure(job)
		}
	}()

	start := time.Now()
	err := handler(wp.ctx, job)
	duration := time.Since(start)

	if err != nil {
		wp.logger.Error("Job processing failed", zap.String("job_id", job.ID), zap.Error(err), zap.Duration("duration", duration))
		wp.handleFailure(job)
		return
	}

	wp.logger.Info("Job completed successfully", zap.String("job_id", job.ID), zap.Duration("duration", duration))
}

func (wp *WorkerPool) handleFailure(job *Job) {
	// Sync backward and forward compatibility fields
	maxRetries := job.MaxRetries
	if maxRetries == 0 {
		maxRetries = job.MaxRetry
	}
	if maxRetries == 0 {
		maxRetries = 3 // default retries
	}

	retryCount := job.RetryCount
	if retryCount == 0 {
		retryCount = job.Retry
	}

	if retryCount >= maxRetries {
		wp.logger.Error("Job exceeded maximum retries, discarding", zap.String("job_id", job.ID), zap.Int("retries_attempted", retryCount))
		return
	}

	retryCount++
	job.RetryCount = retryCount
	job.Retry = retryCount
	
	// Exponential backoff logic: backoff = 2^retryCount * 1 second
	backoffSec := math.Pow(2, float64(retryCount))
	backoffDuration := time.Duration(backoffSec) * time.Second

	wp.logger.Info("Scheduling retry for job",
		zap.String("job_id", job.ID),
		zap.Int("retry_attempt", retryCount),
		zap.Duration("backoff", backoffDuration),
	)

	// Asynchronously enqueue the job after the backoff duration
	go func(j *Job, delay time.Duration) {
		time.Sleep(delay)
		wp.mu.RLock()
		closed := wp.isClosed
		wp.mu.RUnlock()
		
		if closed {
			wp.logger.Warn("Failed to submit retry job: worker pool is closed", zap.String("job_id", j.ID))
			return
		}

		if err := wp.Submit(j); err != nil {
			wp.logger.Error("Failed to re-submit job after backoff", zap.String("job_id", j.ID), zap.Error(err))
		}
	}(job, backoffDuration)
}
