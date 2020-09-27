package scheduler

import (
	"github.com/google/uuid"
	"github.com/stepan-s/jobro/pool/task"
)

type TaskSettings struct {
	Cron  string `json:"cron"`
	Cmd   string `json:"cmd"`
	Group string `json:"group"`
}

type TaskStats struct {
	Done    int64 `json:"done"`
	Running int64 `json:"running"`
	Failed  int64 `json:"failed"`
	Errors  int64 `json:"errors"`
}

type TaskInfo struct {
	Id       uuid.UUID    `json:"id"`
	Settings TaskSettings `json:"settings"`
	Stats    TaskStats    `json:"stats"`
	Pids     []int        `json:"pids"`
}

type CronTask struct {
	Settings TaskSettings
	Stats    TaskStats
	Task     *task.Task
}

func (cronTask CronTask) Run() {
	go cronTask.Task.Exec("scheduled")
}

func (cronTask CronTask) getInfo() TaskInfo {
	return TaskInfo{
		Id:       cronTask.Task.GetId(),
		Settings: cronTask.Settings,
		Stats:    cronTask.Stats,
		Pids:     cronTask.Task.GetPids(),
	}
}
