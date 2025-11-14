package jobs

import (
	. "Topicgram/database"
	"Topicgram/model"
	"Topicgram/services/bots"
	"Topicgram/services/cron"
	"time"

	"gitlab.com/CoiaPrant/clog"
	"gorm.io/gorm/clause"
)

func init() {
	_, err := cron.AddCron("0 0 * * *", VerificationCleanup)
	if err != nil {
		clog.Fatalf("[CronJob] failed to add job, error: %s", err)
		return
	}
}

func VerificationCleanup() {
	err := DB().Model(model.Topic{}).Where("verification", model.VerificationNotCompleted).Not("challange_sent", 0).Where(clause.Lte{Column: "challange_sent", Value: time.Now().Add(bots.CAPTCHA_DURATION).Unix()}).Updates(map[string]any{
		"challange_id":   0,
		"challange_sent": 0,
	}).Error
	if err != nil {
		clog.Errorf("[CronJob][Verification Cleanup] failed to execute, error: %s", err)
		return
	}

	clog.Success("[CronJob][Verification Cleanup] Execute completed")
}
