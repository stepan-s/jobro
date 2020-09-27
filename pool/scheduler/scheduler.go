package scheduler

import (
	"github.com/google/uuid"
	"github.com/robfig/cron"
	"github.com/stepan-s/jobro/log"
	"github.com/stepan-s/jobro/pool/task"
)

type StopCommand struct {
	onstop func()
}

type SetScheduleCommand struct {
	tasks []TaskSettings
}

type RunTaskCommand struct {
	id uuid.UUID
}

type GetInfoCommand struct {
	response chan []TaskInfo
}

type Scheduler struct {
	schedule          []*CronTask
	done              int64
	running           int64
	failed            int64
	errors            int64
	cron              *cron.Cron
	taskNotifications chan task.Notify
	stopChan          chan StopCommand
	setChan           chan SetScheduleCommand
	runChan           chan RunTaskCommand
	getInfoChan       chan GetInfoCommand
}

func New() *Scheduler {
	scheduler := &Scheduler{
		taskNotifications: make(chan task.Notify, 100),
		stopChan:          make(chan StopCommand, 1),
		setChan:           make(chan SetScheduleCommand, 1),
		runChan:           make(chan RunTaskCommand, 100),
		getInfoChan:       make(chan GetInfoCommand, 1),
	}

	// Scheduler main loop
	go func() {
		exit := false
		var onstop func() = nil
	loop:
		for {
			select {
			case event := <-scheduler.taskNotifications:
				cronTask := findCronTaskByUUID(scheduler.schedule, event.Id)
				switch event.Action {
				case task.Start:
					scheduler.running += 1
					if cronTask != nil {
						cronTask.Stats.Running += 1
					}
				case task.Stop:
					scheduler.done += 1
					scheduler.running -= 1
					if cronTask != nil {
						cronTask.Stats.Done += 1
						cronTask.Stats.Running -= 1
					}
					if exit {
						log.Info("Cron tasks in progress: %d", scheduler.running)
						if scheduler.running == 0 {
							log.Info("Scheduler tasks stopped")
							if onstop != nil {
								onstop()
							}
							break loop
						}
					}
				case task.FailStart:
					scheduler.failed += 1
					if cronTask != nil {
						cronTask.Stats.Failed += 1
					}
				case task.Error:
					scheduler.errors += 1
					if cronTask != nil {
						cronTask.Stats.Errors += 1
					}
				}
			case setScheduleCommand := <-scheduler.setChan:
				scheduler.setTasks(setScheduleCommand.tasks)
			case runTaskCommand := <-scheduler.runChan:
				cronTask := findCronTaskByUUID(scheduler.schedule, runTaskCommand.id)
				if cronTask != nil {
					go cronTask.Task.Exec("manual")
				} else {
					log.Error("Task %v not found", runTaskCommand.id)
				}
			case stopCommand := <-scheduler.stopChan:
				log.Info("Scheduler stop tasks")
				exit = true
				if scheduler.cron != nil {
					scheduler.cron.Stop()
				}
				if scheduler.running == 0 {
					log.Info("Scheduler tasks stopped")
					if stopCommand.onstop != nil {
						stopCommand.onstop()
					}
					break loop
				} else {
					onstop = stopCommand.onstop
					for _, cronTask := range scheduler.schedule {
						cronTask.Task.Cancel()
					}
				}
			case getInfoCommand := <-scheduler.getInfoChan:
				getInfoCommand.response <- scheduler.getInfo()
			}
		}
	}()
	return scheduler
}

func (scheduler *Scheduler) GetDone() int64 {
	return scheduler.done
}

func (scheduler *Scheduler) GetRunning() int64 {
	return scheduler.running
}

func (scheduler *Scheduler) GetFailed() int64 {
	return scheduler.failed
}

func (scheduler *Scheduler) GetErrors() int64 {
	return scheduler.errors
}

func (scheduler *Scheduler) SetTasks(tasks []TaskSettings) {
	scheduler.setChan <- SetScheduleCommand{tasks: tasks}
}

func (scheduler *Scheduler) RunTask(id uuid.UUID) {
	scheduler.runChan <- RunTaskCommand{id: id}
}

func (scheduler *Scheduler) Stop(onstop func()) {
	scheduler.stopChan <- StopCommand{onstop: onstop}
}

func (scheduler *Scheduler) setTasks(tasks []TaskSettings) {
	if scheduler.cron != nil {
		scheduler.cron.Stop()
	}
	log.Info("Set scheduler tasks")
	scheduler.cron = cron.New()
	var schedule []*CronTask
	for _, set := range tasks {
		cronTask := findCronTask(scheduler.schedule, set.Cron, set.Cmd)
		if cronTask == nil {
			t := task.New(set.Cmd, scheduler.taskNotifications)
			cronTask = &CronTask{
				Settings: set,
				Task:     &t,
			}
			log.Info("Add task: %v", set)
		} else {
			log.Debug("Task has not changed: %v", set)
		}
		schedule = append(schedule, cronTask)
		if set.Cron != "manual" {
			err := scheduler.cron.AddJob(set.Cron, *cronTask)
			if err != nil {
				log.Error("Fail pass task to cron: %v, error: %v", set, err)
			}
		}
	}
	for _, tsk := range scheduler.schedule {
		if findCronTask(schedule, tsk.Settings.Cron, tsk.Settings.Cmd) == nil {
			log.Info("Remove task: %v", tsk.Settings)
			tsk.Task.Cancel()
		}
	}
	scheduler.schedule = schedule
	scheduler.cron.Start()
}

func findCronTask(schedule []*CronTask, cron string, cmd string) *CronTask {
	for _, cronTask := range schedule {
		if cronTask.Settings.Cron == cron && cronTask.Settings.Cmd == cmd {
			return cronTask
		}
	}
	return nil
}

func findCronTaskByUUID(schedule []*CronTask, id uuid.UUID) *CronTask {
	for _, cronTask := range schedule {
		if cronTask.Task.GetId() == id {
			return cronTask
		}
	}
	return nil
}

func (scheduler *Scheduler) getInfo() []TaskInfo {
	var info []TaskInfo
	for _, tsk := range scheduler.schedule {
		info = append(info, tsk.getInfo())
	}
	return info
}

func (scheduler *Scheduler) GetInfo() []TaskInfo {
	response := make(chan []TaskInfo, 1)
	scheduler.getInfoChan <- GetInfoCommand{
		response: response,
	}
	return <-response
}
