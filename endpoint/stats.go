package endpoint

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stepan-s/jobro/pool/instant"
	"github.com/stepan-s/jobro/pool/scheduler"
	"net/http"
)

func BindMetrics(cronScheduler *scheduler.Scheduler, instantPool *instant.Pools, pattern string) {
	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_schedule_tasks_done",
			Help: "The total number successfully executed tasks",
		}, func() float64 {
			return float64(cronScheduler.GetDone())
		}))
	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_schedule_tasks_failed_start",
			Help: "The total number failed tasks starts",
		}, func() float64 {
			return float64(cronScheduler.GetFailed())
		}))
	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_schedule_tasks_errors",
			Help: "The total number tasks executed with errors",
		}, func() float64 {
			return float64(cronScheduler.GetErrors())
		}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "jobro_schedule_tasks_running",
			Help: "The current number running tasks",
		}, func() float64 {
			return float64(cronScheduler.GetRunning())
		}))

	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_instant_tasks_done",
			Help: "The total number successfully executed tasks",
		}, func() float64 {
			return float64(instantPool.GetDone())
		}))
	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_instant_tasks_failed_start",
			Help: "The total number failed tasks starts",
		}, func() float64 {
			return float64(instantPool.GetFailed())
		}))
	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_instant_tasks_errors",
			Help: "The total number tasks executed with errors",
		}, func() float64 {
			return float64(instantPool.GetErrors())
		}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "jobro_instant_tasks_running",
			Help: "The current number running tasks",
		}, func() float64 {
			return float64(instantPool.GetRunning())
		}))

	http.Handle(pattern, promhttp.Handler())
}
