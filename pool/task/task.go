package task

import (
	"github.com/google/uuid"
	"github.com/mattn/go-shellwords"
	"github.com/stepan-s/jobro/log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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
	if filepath.Base(command) == command {
		lp, err := exec.LookPath(command)
		if err != nil {
			task.state <- Notify{FailStart, pid, task.id}
			log.Error("Fail find command, %s Task: %v, error: %v", execDescription, task.cmd, err)
			return
		}
		command = lp
	}

	log.Debug("Start process %v, with: %v", command, args)
	proc, err := os.StartProcess(command, args, &os.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			nil,
			nil,
		},
		Sys: &syscall.SysProcAttr{Noctty: true},
	})
	if err != nil {
		task.state <- Notify{FailStart, 0, task.id}
		log.Error("Fail start %s Task: %v, error: %v", execDescription, task.cmd, err)
		return
	}

	pid = proc.Pid
	task.pids = append(task.pids, pid)
	task.state <- Notify{Start, pid, task.id}

	err = proc.Release()
	if err != nil {
		log.Error("Fail release process pid %d, error: %v", pid, err)
	}

	proc, err = os.FindProcess(pid)
	if err == nil {
		log.Info("Task %v %s exec %v", pid, execDescription, task.cmd)
		state, err := proc.Wait()
		if err == nil {
			exitCode := state.ExitCode()
			if exitCode == 0 {
				log.Info("Task %v done", pid)
			} else {
				task.state <- Notify{Error, pid, task.id}
				log.Info("Task %v fail with code: %v", pid, exitCode)
			}
		} else {
			log.Info("Task wait %v fail with error: %v", pid, err)
		}
	} else {
		log.Info("Task %v not found, error: %v", pid, err)
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
