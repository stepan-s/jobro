package task

import (
	"github.com/google/uuid"
	"github.com/mattn/go-shellwords"
	"github.com/stepan-s/jobro/log"
	"os"
	"os/exec"
)

const FailStart = 0
const Error = 2
const Start = 1
const Stop = -1

type Notify struct {
	Action int
	Pid    int
	Id     uuid.UUID
}

type Task struct {
	cmd   string
	id    uuid.UUID
	state chan Notify
	pids  []int
}

func New(cmd string, notifyChannel chan Notify) Task {
	return Task{cmd, uuid.New(), notifyChannel, []int{}}
}

func (task *Task) GetId() uuid.UUID {
	return task.id
}

func (task *Task) GetCmd() string {
	return task.cmd
}

func (task *Task) GetRunning() int {
	return len(task.pids)
}

func (task *Task) Exec(execDescription string) {
	pid := 0

	defer func() {
		if pid != 0 {
			task.state <- Notify{Stop, pid, task.id}
			var pids []int
			for _, p := range task.pids {
				if p != pid {
					pids = append(pids, p)
				}
			}
			task.pids = pids
			pid = 0
		}
	}()

	var err error
	var args []string
	args, err = shellwords.Parse(task.cmd)
	if err != nil {
		task.state <- Notify{FailStart, pid, task.id}
		log.Error("Fail parse args, %s Task: %v, error: %v", execDescription, task.cmd, err)
		return
	}

	command := args[0]
	cmd := exec.Command(command, args[1:]...);

	log.Debug("Start process %v, with: %v", command, args)
	err = cmd.Start()
	if err != nil {
		task.state <- Notify{FailStart, 0, task.id}
		log.Error("Fail start %s Task: %v, error: %v", execDescription, task.cmd, err)
		return
	}

	pid = cmd.Process.Pid
	task.pids = append(task.pids, pid)
	task.state <- Notify{Start, pid, task.id}

	log.Info("Task %v %s exec %v", pid, execDescription, task.cmd)
	err = cmd.Wait()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			task.state <- Notify{Error, pid, task.id}
			log.Info("Task %v fail with code: %v", pid, e.ExitCode())
		} else {
			log.Info("Task wait %v fail with error: %v", pid, err)
		}
	} else {
		log.Info("Task %v done", pid)
	}
}

func (task *Task) Cancel() {
	for _, pid := range task.pids {
		process, err := os.FindProcess(pid)
		if err == nil {
			err = process.Signal(os.Interrupt)
			if err != nil {
				log.Error("Fail interrupt pid %d, error: %v", pid, err)
			} else {
				log.Info("Interrupt pid %d", pid)
			}
		} else {
			log.Error("Fail interrupt pid %d, error: %v", pid, err)
		}
	}
}

func (task *Task) CancelLimited(limit int) {
	for _, pid := range task.pids {
		process, err := os.FindProcess(pid)
		if err == nil {
			err = process.Signal(os.Interrupt)
			if err != nil {
				log.Error("Fail interrupt pid %d, error: %v", pid, err)
			} else {
				log.Info("Interrupt pid %d", pid)
			}
		} else {
			log.Error("Fail interrupt pid %d, error: %v", pid, err)
		}
		limit -= 1
		if limit <= 0 {
			break
		}
	}
}

func (task *Task) GetPids() []int {
	return task.pids
}
