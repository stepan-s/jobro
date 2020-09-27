package instant

import (
	"github.com/stepan-s/jobro/log"
)

type SetTasksCommand struct {
	settings []PoolSettings
}

type PoolsStopCommand struct {
	onstop func()
}

type PoolsGetInfoCommand struct {
	response chan []PoolInfo
}

type Pools struct {
	items             []*Pool
	running           int
	setTasksChan      chan SetTasksCommand
	stopChan          chan PoolsStopCommand
	getInfoChan       chan PoolsGetInfoCommand
	poolNotifications chan PoolNotify
}

func New() *Pools {
	pools := &Pools{
		setTasksChan:      make(chan SetTasksCommand, 1),
		stopChan:          make(chan PoolsStopCommand, 1),
		getInfoChan:       make(chan PoolsGetInfoCommand, 1),
		poolNotifications: make(chan PoolNotify, 100),
	}

	// main loop
	go func() {
		exit := false
		var onstop func() = nil
	loop:
		for {
			select {
			case event := <-pools.poolNotifications:
				switch event.Action {
				case PoolStart:
					pools.running += 1
				case PoolStop:
					pools.running -= 1
					pools.remove(event.Pool)
					if exit {
						log.Info("Instant pools active: %d", pools.running)
						if pools.running == 0 {
							log.Info("Instant pools stopped")
							if onstop != nil {
								onstop()
							}
							break loop
						}
					}
				}
			case setTasksCommand := <-pools.setTasksChan:
				pools.setTasks(setTasksCommand.settings)
			case getInfoCommand := <-pools.getInfoChan:
				getInfoCommand.response <- pools.getInfo()
			case stopCommand := <-pools.stopChan:
				log.Info("Instant pools stop")
				exit = true
				if pools.running == 0 {
					log.Info("Instant pools stopped")
					if stopCommand.onstop != nil {
						stopCommand.onstop()
					}
					break loop
				} else {
					onstop = stopCommand.onstop
					for _, pool := range pools.items {
						pool.Stop()
					}
				}
			}
		}
	}()
	return pools
}

func (pools *Pools) SetTasks(settings []PoolSettings) {
	pools.setTasksChan <- SetTasksCommand{settings: settings}
}

func (pools *Pools) Stop(onstop func()) {
	pools.stopChan <- PoolsStopCommand{onstop: onstop}
}

func (pools *Pools) setTasks(settings []PoolSettings) {
	var newPools []*Pool
	for _, set := range settings {
		pool := findPoolByCmd(pools.items, set.Cmd)
		if pool != nil {
			newPools = append(newPools, pool)
			pool.SetCount(set.Count)
			log.Info("Set count %d for instant pool %v", set.Count, set.Cmd)
		} else {
			pool = NewPool(set, pools.poolNotifications)
			newPools = append(newPools, pool)
			log.Info("Add instant pool %v count %d", set.Cmd, set.Count)
			pool.Start()
		}
	}
	for _, pool := range pools.items {
		exist := findPoolByCmd(newPools, pool.Settings.Cmd)
		if exist == nil {
			newPools = append(newPools, pool)
			pool.Stop()
			log.Info("Stop instant pool %v", pool.Settings.Cmd)
		}
	}
	pools.items = newPools
}

func (pools *Pools) remove(pool *Pool) {
	var newPools []*Pool
	for _, p := range pools.items {
		if p.Settings.Cmd != pool.Settings.Cmd {
			newPools = append(newPools, p)
		}
	}
	pools.items = newPools
}

func findPoolByCmd(list []*Pool, cmd string) *Pool {
	for _, pool := range list {
		if pool.Settings.Cmd == cmd {
			return pool
		}
	}
	return nil
}

func (pools *Pools) GetDone() int64 {
	var total int64
	for _, pool := range pools.items {
		total += pool.Stats.Done
	}
	return total
}

func (pools *Pools) GetRunning() int64 {
	var total int64
	for _, pool := range pools.items {
		total += pool.Stats.Running
	}
	return total
}

func (pools *Pools) GetFailed() int64 {
	var total int64
	for _, pool := range pools.items {
		total += pool.Stats.Failed
	}
	return total
}

func (pools *Pools) GetErrors() int64 {
	var total int64
	for _, pool := range pools.items {
		total += pool.Stats.Errors
	}
	return total
}

func (pools *Pools) getInfo() []PoolInfo {
	var info []PoolInfo
	for _, pool := range pools.items {
		info = append(info, pool.getInfo())
	}
	return info
}

func (pools *Pools) GetInfo() []PoolInfo {
	response := make(chan []PoolInfo, 1)
	pools.getInfoChan <- PoolsGetInfoCommand{
		response: response,
	}
	return <-response
}
