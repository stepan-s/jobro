package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/mattn/go-shellwords"
	"github.com/stepan-s/jobro/log"
	"github.com/stepan-s/jobro/pool/instant"
	"github.com/stepan-s/jobro/pool/scheduler"
	"os/exec"
)

type Config struct {
	command     string
	fingerprint string
	onUpdate    func(*TasksConfig)
}

type TasksConfig struct {
	Schedule []scheduler.TaskSettings
	Instant  []instant.PoolSettings
}

func New(command string) Config {
	return Config{
		command: command,
	}
}

func (config *Config) SetOnUpdate(onUpdate func(*TasksConfig))  {
	config.onUpdate = onUpdate
}

func (config *Config) Update() bool {
	var err error
	var args []string
	args, err = shellwords.Parse(config.command)
	if err != nil {
		log.Error("Fail parse args: %v, error: %v", config.command, err)
		return false
	}

	log.Debug("Execute config command: '%v' with args: %v", args[0], args[1:])
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		log.Error("Fail get config %v", err)
		return false
	}

	hash := sha256.New()
	hash.Write(out)
	fingerprint := fmt.Sprintf("%x", hash.Sum(nil))
	if fingerprint == config.fingerprint {
		log.Info("Config not changed")
		return true
	}

	var conf TasksConfig
	err = json.Unmarshal(out, &conf)
	if err != nil {
		log.Error("Fail parse config: %v", err)
		return false
	}

	if config.onUpdate != nil {
		config.onUpdate(&conf)
	}
	return true
}
