package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// JobFunc is the signature for background jobs
type JobFunc func(ctx context.Context)

// BackgroundJob represents a single background task
type BackgroundJob struct {
	Name     string
	Interval time.Duration
	Func     JobFunc
	Delay    time.Duration // Optional initial delay
}

// BackgroundJobManager manages the lifecycle of multiple background jobs
type BackgroundJobManager struct {
	logger *zap.Logger
	jobs   []BackgroundJob
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewBackgroundJobManager creates a new BackgroundJobManager
func NewBackgroundJobManager(logger *zap.Logger) *BackgroundJobManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &BackgroundJobManager{
		logger: logger,
		jobs:   make([]BackgroundJob, 0),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Register adds a new background job to the manager
func (m *BackgroundJobManager) Register(name string, interval time.Duration, fn JobFunc) {
	m.jobs = append(m.jobs, BackgroundJob{
		Name:     name,
		Interval: interval,
		Func:     fn,
		Delay:    0,
	})
}

// RegisterWithDelay adds a new background job with an initial delay
func (m *BackgroundJobManager) RegisterWithDelay(name string, interval, delay time.Duration, fn JobFunc) {
	m.jobs = append(m.jobs, BackgroundJob{
		Name:     name,
		Interval: interval,
		Func:     fn,
		Delay:    delay,
	})
}

// StartAll starts all registered background jobs
func (m *BackgroundJobManager) StartAll() {
	m.logger.Info("Starting all background jobs", zap.Int("count", len(m.jobs)))

	for _, job := range m.jobs {
		m.wg.Add(1)
		go m.runJob(job)
	}
}

// StopAll stops all background jobs and waits for them to finish
func (m *BackgroundJobManager) StopAll() {
	m.logger.Info("Stopping all background jobs...")
	m.cancel()
	m.wg.Wait()
	m.logger.Info("All background jobs stopped successfully")
}

func (m *BackgroundJobManager) runJob(job BackgroundJob) {
	defer m.wg.Done()

	if job.Delay > 0 {
		m.logger.Debug("Delaying job start", zap.String("job", job.Name), zap.Duration("delay", job.Delay))
		select {
		case <-time.After(job.Delay):
			// Proceed
			job.Func(m.ctx) // Run once after delay if desired, or let ticker handle it. We'll run it once here.
		case <-m.ctx.Done():
			return
		}
	}

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	m.logger.Info("Background job started", zap.String("job", job.Name), zap.Duration("interval", job.Interval))

	for {
		select {
		case <-ticker.C:
			m.logger.Debug("Executing background job", zap.String("job", job.Name))
			job.Func(m.ctx)
		case <-m.ctx.Done():
			m.logger.Info("Background job stopped", zap.String("job", job.Name))
			return
		}
	}
}
