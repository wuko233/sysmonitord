package timer

import (
	"sync"
	"sysmonitord/pkg/logger"
	"time"

	"go.uber.org/zap"
)

type Job interface {
	Run() error
	Name() string
}

type Scheduler struct {
	ticker   *time.Ticker
	stopCh   chan struct{}
	job      Job
	wg       sync.WaitGroup
	interval time.Duration
}

func NewScheduler(interval time.Duration, job Job) *Scheduler {
	return &Scheduler{
		ticker:   time.NewTicker(interval),
		stopCh:   make(chan struct{}),
		job:      job,
		interval: interval,
	}
}

func (s *Scheduler) Start() {
	logger.Log.Info("[monitor] 定时任务已启动", zap.String("job", s.job.Name()), zap.Duration("interval", s.interval))

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		logger.Log.Info("[monitor] 执行定时任务", zap.String("job", s.job.Name()))
		if err := s.job.Run(); err != nil {
			logger.Log.Error("[monitor] 定时任务执行失败", zap.String("job", s.job.Name()), zap.Error(err))
		}

		for {
			select {
			case <-s.ticker.C:
				logger.Log.Info("[monitor] 执行定时任务", zap.String("job", s.job.Name()))
				if err := s.job.Run(); err != nil {
					logger.Log.Error("[monitor] 定时任务执行失败", zap.String("job", s.job.Name()), zap.Error(err))
				}
			case <-s.stopCh:
				logger.Log.Info("[monitor] 定时任务已停止", zap.String("job", s.job.Name()))
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.ticker.Stop()
	s.wg.Wait()
}
