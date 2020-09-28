package endpoint

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stepan-s/jobro/pool/instant"
	"github.com/stepan-s/jobro/pool/scheduler"
	"net/http"
)

const SubjectReload = 0
const ActionIncrement = 0

type StatsTransaction struct {
	Subject uint8
	Action  uint8
	Value	uint64
}

type Stats struct {
	inChan  chan StatsTransaction
	reloads uint64
}

func NewStats() *Stats {
	stats := &Stats{
		inChan:  make(chan StatsTransaction, 100),
		reloads: 0,
	}
	go func() {
		for {
			select {
			case transaction := <- stats.inChan:
				switch transaction.Subject {
				case SubjectReload:
					if transaction.Action == ActionIncrement {
						stats.reloads += transaction.Value
					}
				}
			}
		}
	}()
	return stats
}

func (stats *Stats) Send(transaction StatsTransaction) {
	stats.inChan <- transaction
}

func BindMetrics(cronScheduler *scheduler.Scheduler, instantPool *instant.Pools, stats *Stats, pattern string) {
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

	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "jobro_reloads",
			Help: "The total number config reloads",
		}, func() float64 {
			return float64(stats.reloads)
		}))

	http.Handle(pattern, promhttp.Handler())
}
