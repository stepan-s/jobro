package endpoint

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stepan-s/jobro/config"
	"github.com/stepan-s/jobro/log"
	"github.com/stepan-s/jobro/pool/instant"
	"github.com/stepan-s/jobro/pool/scheduler"
	"net/http"
)

type Info struct {
	Schedule []scheduler.TaskInfo `json:"schedule"`
	Instant  []instant.PoolInfo   `json:"instant"`
}

func BindApi(cronScheduler *scheduler.Scheduler, instantPool *instant.Pools, conf *config.Config, pattern string) {
	http.HandleFunc(pattern+"/info", func(w http.ResponseWriter, r *http.Request) {
		info := Info{
			Schedule: cronScheduler.GetInfo(),
			Instant:  instantPool.GetInfo(),
		}

		res, err := json.Marshal(info)
		if err != nil {
			log.Error("Fail prepare json: %v", err)
			w.Header().Add("X-Error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		_, err2 := w.Write(res)
		if err2 != nil {
			log.Error("Fail sign auth: %v", err)
			w.Header().Add("X-Error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	http.HandleFunc(pattern+"/reload", func(w http.ResponseWriter, r *http.Request) {
		conf.Update()
	})

	http.HandleFunc(pattern+"/schedule/run", func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.URL.Query().Get("id"))
		if err != nil {
			w.Header().Add("X-Error", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cronScheduler.RunTask(id)
	})
}
