package instant

import (
	"github.com/google/uuid"
	"github.com/stepan-s/jobro/log"
	"github.com/stepan-s/jobro/pool/task"
	"time"
)

const PoolStart = 1
const PoolStop = -1

type PoolSettings struct {
	Cmd   string `json:"cmd"`
	Count int    `json:"count"`
	Group string `json:"group"`
}

type PoolStats struct {
	Done    int64 `json:"done"`
	Running int64 `json:"running"`
	Failed  int64 `json:"failed"`
	Errors  int64 `json:"errors"`
}

type PoolInfo struct {
	Id       uuid.UUID    `json:"id"`
	Settings PoolSettings `json:"settings"`
	Stats    PoolStats    `json:"stats"`
	Pids     []int        `json:"pids"`
}

type PoolNotify struct {
	Action int
	Pool   *Pool
}

type PoolStartCommand struct{}
type PoolStopCommand struct{}
type PoolCountCommand struct {
	count int
}

type Pool struct {
	Settings          PoolSettings
	Stats             PoolStats
	Workers           *task.Task
	state             chan PoolNotify
	taskNotifications chan task.Notify
	startChan         chan PoolStartCommand
	stopChan          chan PoolStopCommand
	setCountChan      chan PoolCountCommand
}

func NewPool(set PoolSettings, state chan PoolNotify) *Pool {
	taskNotifications := make(chan task.Notify, 100)
	worker := task.New(set.Cmd, taskNotifications)
	pool := &Pool{
		Settings:          set,
		Workers:           &worker,
		state:             state,
		taskNotifications: taskNotifications,
		startChan:         make(chan PoolStartCommand, 1),
		stopChan:          make(chan PoolStopCommand, 1),
		setCountChan:      make(chan PoolCountCommand, 100),
	}

	// main loop
	go func() {
		exit := false
	loop:
		for {
			select {
			case event := <-pool.taskNotifications:
				switch event.Action {
				case task.Start:
					pool.Stats.Running += 1
				case task.Stop:
					pool.Stats.Done += 1
					pool.Stats.Running -= 1
					if exit {
						log.Info("Cron tasks in progress: %d", pool.Stats.Running)
						if pool.Stats.Running == 0 {
							log.Info("Instant pool stopped")
							pool.state <- PoolNotify{PoolStop, pool}
							break loop
						}
					} else {
						if pool.Stats.Running < int64(pool.Settings.Count) {
							go pool.Workers.Exec("instant")
						}
					}
				case task.FailStart:
					pool.Stats.Failed += 1
					go func() {
						time.Sleep(time.Duration(5) * time.Second)
						if pool.Stats.Running < int64(pool.Settings.Count) {
							go pool.Workers.Exec("instant")
						}
					}()
				case task.Error:
					pool.Stats.Errors += 1
				}
			case <-pool.startChan:
				for i := pool.Stats.Running; i < int64(pool.Settings.Count); i += 1 {
					go pool.Workers.Exec("instant")
				}
				pool.state <- PoolNotify{PoolStart, pool}
			case setCountCommand := <-pool.setCountChan:
				pool.setCount(setCountCommand.count)
			case <-pool.stopChan:
				log.Info("Instant pool stop tasks")
				exit = true
				if pool.Stats.Running == 0 {
					log.Info("Instant pool stopped")
					pool.state <- PoolNotify{PoolStop, pool}
					break loop
				} else {
					pool.Workers.Cancel()
				}
			}
		}
	}()
	return pool
}

func (pool *Pool) SetCount(count int) {
	pool.setCountChan <- PoolCountCommand{
		count: count,
	}
}

func (pool *Pool) setCount(count int) {
	pool.Settings.Count = count
	running := pool.Workers.GetRunning()
	if running < count {
		for i := running; i < count; i += 1 {
			go pool.Workers.Exec("instant")
		}
	} else if running > count {
		pool.Workers.CancelLimited(running - count)
	}
}

func (pool *Pool) Start() {
	pool.startChan <- PoolStartCommand{}
}

func (pool *Pool) Stop() {
	pool.stopChan <- PoolStopCommand{}
}

func (pool *Pool) GetDone() int64 {
	return pool.Stats.Done
}

func (pool *Pool) GetRunning() int64 {
	return pool.Stats.Running
}

func (pool *Pool) GetFailed() int64 {
	return pool.Stats.Failed
}

func (pool *Pool) GetErrors() int64 {
	return pool.Stats.Errors
}

func (pool *Pool) getInfo() PoolInfo {
	return PoolInfo{
		Id:       pool.Workers.GetId(),
		Settings: pool.Settings,
		Stats:    pool.Stats,
		Pids:     pool.Workers.GetPids(),
	}
}
