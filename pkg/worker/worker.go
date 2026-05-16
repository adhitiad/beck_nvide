package worker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
)

// JobType represents the type of background job
type JobType string

const (
	JobVideoTranscoding  JobType = "video_transcoding"
	JobStreamArchive     JobType = "stream_archive"
	JobDailyReport      JobType = "daily_report"
	JobNotificationBatch JobType = "notification_batch"
)

// Job represents a background task
type Job struct {
	ID        string          `json:"id"`
	Type      JobType         `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Priority  int             `json:"priority"`
	Retry     int             `json:"retry"`
	MaxRetry  int             `json:"max_retry"`
	CreatedAt time.Time       `json:"created_at"`
}

// Handler defines the function signature for processing a job
type Handler func(ctx context.Context, job *Job) error

// Pool manages a group of workers processing jobs from a queue
type Pool struct {
	workerCount int
	jobQueue    chan *Job
	handlers    map[JobType]Handler
	logger      *zap.Logger
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPool creates a new worker pool
func NewPool(workerCount int, logger *zap.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workerCount: workerCount,
		jobQueue:    make(chan *Job, 100),
		handlers:    make(map[JobType]Handler),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// RegisterHandler registers a handler for a specific job type
func (p *Pool) RegisterHandler(jobType JobType, handler Handler) {
	p.handlers[jobType] = handler
}

// Enqueue adds a job to the queue
func (p *Pool) Enqueue(job *Job) {
	select {
	case p.jobQueue <- job:
		p.logger.Debug("Job enqueued", zap.String("job_id", job.ID), zap.String("type", string(job.Type)))
	default:
		p.logger.Warn("Job queue full, dropping job", zap.String("job_id", job.ID))
	}
}

// Start launches the workers
func (p *Pool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("Worker pool started", zap.Int("workers", p.workerCount))
}

// Stop gracefully shuts down the pool
func (p *Pool) Stop() {
	p.cancel()
	close(p.jobQueue)
	
	// Wait for workers to finish with a timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker pool stopped gracefully")
	case <-time.After(30 * time.Second):
		p.logger.Warn("Worker pool forced shutdown after timeout")
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	p.logger.Debug("Worker started", zap.Int("worker_id", id))

	for job := range p.jobQueue {
		p.processJob(job)
	}
}

func (p *Pool) processJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("Job processing panicked", zap.String("job_id", job.ID), zap.Any("panic", r))
		}
	}()

	handler, ok := p.handlers[job.Type]
	if !ok {
		p.logger.Error("No handler registered for job type", zap.String("type", string(job.Type)))
		return
	}

	p.logger.Info("Processing job", zap.String("job_id", job.ID), zap.String("type", string(job.Type)))
	
	err := handler(p.ctx, job)
	if err != nil {
		p.logger.Error("Job failed", zap.String("job_id", job.ID), zap.Error(err))
		
		if job.Retry < job.MaxRetry {
			job.Retry++
			backoff := time.Duration(job.Retry*job.Retry) * time.Second
			p.logger.Info("Retrying job", zap.String("job_id", job.ID), zap.Int("retry", job.Retry), zap.Duration("backoff", backoff))
			
			go func(j *Job, b time.Duration) {
				time.Sleep(b)
				p.Enqueue(j)
			}(job, backoff)
		} else {
			p.logger.Warn("Job reached max retries, moving to DLQ (simulated)", zap.String("job_id", job.ID))
		}
		return
	}

	p.logger.Info("Job completed successfully", zap.String("job_id", job.ID))
}
