package cron

import (
	"github.com/robfig/cron/v3"
	"gitlab.com/CoiaPrant/clog"
)

var (
	logger  = cron.PrintfLogger(clog.Native("[CronJob]", clog.LevelDebug))
	cronjob = cron.New(cron.WithChain(cron.Recover(logger), cron.SkipIfStillRunning(logger)))
)

func Start() {
	cronjob.Start()
	clog.Info("[CronJob] Start all cron jobs")

}

func Stop() {
	cronjob.Stop()
	clog.Info("[CronJob] Stop all cron jobs")
}

func AddCron(spec string, cmd func()) (id cron.EntryID, err error) {
	id, err = cronjob.AddFunc(spec, cmd)
	if err != nil {
		clog.Errorf("[CronJob] failed to add job, spec: %s, error: %s", spec, err)
		return
	}

	clog.Debugf("[CronJob] Added job, spec: %s, id: %d", spec, id)
	return
}

func RemoveCron(id cron.EntryID) {
	cronjob.Remove(id)
	clog.Debugf("[CronJob] Removed job, id: %d", id)
}
