package worker

import (
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"sync"
	"time"
)

var ErrNoWorkers = errors.New("attempting to create worker pool with less than 1 worker")
var ErrNegativeChannelSize = errors.New("attempting to create worker pool with a negative channel size")

type Pool interface {
	Start()
	Stop()
	AddWork(Task)
}

type Task interface {
	Execute() error
	OnFailure(error)
}

// WorkerPool processed workers
type WorkerPool struct {
	numWorkers int
	tasks      chan Task
	start      sync.Once
	stop       sync.Once
	quit       chan struct{}
}

func NewWorkerPool(numWorkers int, channelSize int) (*WorkerPool, error) {
	if numWorkers <= 0 {
		return nil, ErrNoWorkers
	}
	if channelSize < 0 {
		return nil, ErrNegativeChannelSize
	}

	tasks := make(chan Task, channelSize)

	return &WorkerPool{
		numWorkers: numWorkers,
		tasks:      tasks,

		start: sync.Once{},
		stop:  sync.Once{},

		quit: make(chan struct{}),
	}, nil
}

func (wp *WorkerPool) Start() {
	wp.start.Do(func() {
		logger.Log.Info("worker pool: starting pool")
		wp.startWorkers()
	})
}

func (wp *WorkerPool) Stop() {
	wp.stop.Do(func() {
		logger.Log.Info("worker pool: stopping pool")
		close(wp.quit)
	})
}

// AddWork puts task to queue
func (wp *WorkerPool) AddWork(t Task) {
	select {
	case wp.tasks <- t:
	case <-wp.quit:
	}
}

// AddWorkNonBlocking puts a task in a queue without blocking the calling code
func (wp *WorkerPool) AddWorkNonBlocking(t Task) {
	go wp.AddWork(t)
}

func (wp *WorkerPool) startWorkers() {
	for i := 0; i < wp.numWorkers; i++ {
		go func(workerNum int) {
			logger.Log.Info(fmt.Sprintf("worker pool: starting worker number %d", workerNum))

			for {
				select {
				case <-wp.quit:
					logger.Log.Info(fmt.Sprintf("worker pool: stopping worker %d with quit channel\n", workerNum))
					return
				case task, ok := <-wp.tasks:
					if !ok {
						logger.Log.Info(fmt.Sprintf("worker pool: stopping worker %d with closed tasks channel\n", workerNum))
						return
					}
					logger.Log.Info(fmt.Sprintf("worker pool: worker #%d starts to execute tasks", workerNum))
					if err := task.Execute(); err != nil {
						logger.Log.Error(fmt.Sprintf("worker pool: worker #%d failed", workerNum), zap.Error(err))
						task.OnFailure(err)
					}
				}
			}
		}(i)
	}
}

// WaitingPool processed workers
type WaitingPool struct {
	numWorkers int
	awaitTime  time.Duration
	tasks      chan Task
	start      sync.Once
	stop       sync.Once
	quit       chan struct{}
}

func NewWaitingPool(numWorkers int, channelSize int, awaitTime time.Duration) (*WaitingPool, error) {
	if numWorkers <= 0 {
		return nil, ErrNoWorkers
	}
	if channelSize < 0 {
		return nil, ErrNegativeChannelSize
	}

	tasks := make(chan Task, channelSize)

	return &WaitingPool{
		numWorkers: numWorkers,
		awaitTime:  awaitTime,
		tasks:      tasks,

		start: sync.Once{},
		stop:  sync.Once{},

		quit: make(chan struct{}),
	}, nil
}

func (wp *WaitingPool) Start() {
	wp.start.Do(func() {
		logger.Log.Info("waiting pool: starting pool")
		wp.startWorkers()
	})
}

func (wp *WaitingPool) Stop() {
	wp.stop.Do(func() {
		logger.Log.Info("waiting pool: stopping pool")
		close(wp.quit)
	})
}

// AddWork puts task to queue
func (wp *WaitingPool) AddWork(t Task) {
	select {
	case wp.tasks <- t:
	case <-wp.quit:
	}
}

// AddWorkNonBlocking puts a task in a queue without blocking the calling code
func (wp *WaitingPool) AddWorkNonBlocking(t Task) {
	go wp.AddWork(t)
}

func (wp *WaitingPool) startWorkers() {
	for i := 0; i < wp.numWorkers; i++ {
		go func(workerNum int) {
			logger.Log.Info(fmt.Sprintf("Waiting pool: starting worker number %d", workerNum))

			tick := time.NewTicker(wp.awaitTime)

			for {
				select {
				case <-wp.quit:
					logger.Log.Info(fmt.Sprintf("waiting pool: stopping worker %d with quit channel\n", workerNum))
					return
				case <-tick.C:
					select {
					case task, ok := <-wp.tasks:
						if !ok {
							logger.Log.Info(fmt.Sprintf("waiting pool: stopping worker %d with closed tasks channel\n", workerNum))
							return
						}
						logger.Log.Info(fmt.Sprintf("waiting pool: worker #%d starts to execute tasks", workerNum))
						if err := task.Execute(); err != nil {
							logger.Log.Error(fmt.Sprintf("waiting pool: worker #%d failed", workerNum), zap.Error(err))
							task.OnFailure(err)
						}
					default:
						continue
					}
				}
			}
		}(i)
	}
}
