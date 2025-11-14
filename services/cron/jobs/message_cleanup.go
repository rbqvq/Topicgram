package jobs

import (
	. "Topicgram/database"
	"Topicgram/model"
	"Topicgram/services/cron"
	"time"

	"gitlab.com/CoiaPrant/clog"
	"gorm.io/gorm/clause"
)

func init() {
	_, err := cron.AddCron("0 0 * * *", MessageCleanup)
	if err != nil {
		clog.Fatalf("[CronJob] failed to add job, error: %s", err)
		return
	}
}

func MessageCleanup() {
	err := DB().Model(model.Msg{}).Where(clause.Lte{Column: "created_at", Value: time.Now().AddDate(0, 0, -30)}).Delete(nil).Error
	if err != nil {
		clog.Errorf("[CronJob][Message Cleanup] failed to execute, error: %s", err)
		return
	}

	clog.Success("[CronJob][Message Cleanup] Execute completed")
}
