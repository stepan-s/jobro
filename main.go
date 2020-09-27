package main

import (
	"context"
	"flag"
	"github.com/stepan-s/jobro/config"
	"github.com/stepan-s/jobro/endpoint"
	"github.com/stepan-s/jobro/log"
	"github.com/stepan-s/jobro/pool/instant"
	"github.com/stepan-s/jobro/pool/scheduler"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var addr = flag.String("addr", "localhost:80", "http service address")
	var configCommand = flag.String("config-command", "cat jobro.json", "command that return config")
	var logLevel = flag.Int64("log-level", log.DEBUG, "log level")
	var shutdownTimeout = flag.Int64("shutdown-timeout", 60, "shutdown timeout")
	flag.Parse()

	var logLevelValue = uint8(*logLevel)
	log.Init(os.Stdout, logLevelValue)
	log.Info("Starting")

	log.Info("Options:")
	log.Info("  addr: %v", *addr)
	log.Info("  config-command: %v", *configCommand)
	log.Info("  shutdown-timeout: %v", *shutdownTimeout)
	log.Info("  log-level: %v", *logLevel)

	// Create and run services
	cronScheduler := scheduler.New()
	instantPool := instant.New()
	endpoint.BindApi(cronScheduler, instantPool, "/api")
	endpoint.BindMetrics(cronScheduler, instantPool, "/metrics")
	srv := &http.Server{Addr: *addr}

	exit := make(chan int, 10)

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Emergency("Http server error: %v", err)
		}
		log.Info("Http server stopped")
		if err != http.ErrServerClosed {
			exit <- 1
		} else {
			exit <- 0
		}
	}()

	// Handle shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		log.Info("Interrupt signal received, shutdown begin")

		cronScheduler.Stop(func() {
			exit <- 0
		})
		instantPool.Stop(func() {
			exit <- 0
		})

		// We received an interrupt signal, shut down.
		err := srv.Shutdown(context.Background())
		if err != nil {
			// Error from closing listeners, or context timeout:
			log.Error("Http server shutdown: %v", err)
		}

		select {
		case <-time.After(time.Duration(*shutdownTimeout) * time.Second):
			log.Error("Shutdown timeout reached")
			os.Exit(255)
		}
	}()

	conf := config.New(*configCommand)
	onUpdate := func(taskConfig *config.TasksConfig) {
		cronScheduler.SetTasks(taskConfig.Schedule)
		instantPool.SetTasks(taskConfig.Instant)
	}
	conf.Update(onUpdate)

	go func() {
		sigusr1 := make(chan os.Signal, 1)
		signal.Notify(sigusr1, syscall.SIGUSR1)

		for {
			select {
			case <-sigusr1:
				log.Info("Reload signal received")
				conf.Update(onUpdate)
			}
		}
	}()

	// Wait for stop all services
	services := 3
	exitCode := 0
	for {
		select {
		case status := <-exit:
			exitCode |= status
			services -= 1
			if services <= 0 {
				log.Info("Shutdown completed, exit code: %d", exitCode)
				os.Exit(exitCode)
			}
		}
	}
}
