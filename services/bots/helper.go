package bots

import (
	. "Topicgram/database"
	"Topicgram/model"
	"strings"

	botapi "github.com/OvyFlash/telegram-bot-api"
)

func isServiceMessage(msg *botapi.Message) bool {
	return (msg.NewChatTitle != "" ||
		msg.NewChatPhoto != nil ||
		msg.DeleteChatPhoto ||
		msg.GroupChatCreated || msg.SuperGroupChatCreated || msg.ChannelChatCreated ||
		msg.NewChatMembers != nil || msg.LeftChatMember != nil ||
		msg.PinnedMessage != nil ||
		msg.MessageAutoDeleteTimerChanged != nil ||
		msg.ConnectedWebsite != "" ||
		msg.SuccessfulPayment != nil ||
		msg.WriteAccessAllowed != nil ||
		msg.ForumTopicEdited != nil ||
		msg.GeneralForumTopicHidden != nil || msg.GeneralForumTopicUnhidden != nil ||
		msg.VideoChatScheduled != nil || msg.VideoChatStarted != nil || msg.VideoChatEnded != nil || msg.VideoChatParticipantsInvited != nil)
}

func isIgnoreMessage(msg *botapi.Message) bool {
	return (msg.WebAppData != nil || msg.ReplyMarkup != nil || msg.ForumTopicCreated != nil)
}

func isAllowedMessage(msg *botapi.Message) bool {
	return (msg.Text != "" ||
		msg.Animation != nil || msg.PremiumAnimation != nil ||
		msg.Audio != nil || msg.Document != nil ||
		msg.Photo != nil || msg.Sticker != nil ||
		msg.Video != nil || msg.VideoNote != nil ||
		msg.Voice != nil)
}

func isUnauthorized(err *botapi.Error) bool {
	return err.Code == 401 || err.Code == 404
}

func isBlocked(err *botapi.Error) bool {
	return strings.Contains(err.Message, "bot was blocked by the user")
}

func saveTopic(topic *model.Topic) error {
	if topic.Id == 0 {
		return DB().Create(topic).Error
	}

	return DB().Save(topic).Error
}

func banTopic(topic *model.Topic) error {
	topic.IsBan = true
	topic.Verification = model.VerificationNotSent
	topic.ChallangeId = 0
	topic.ChallangeSent = 0

	if topic.Id == 0 {
		return DB().Create(topic).Error
	}

	return DB().Save(topic).Error
}

func unbanTopic(topic *model.Topic) error {
	topic.IsBan = false

	if topic.TopicId == 0 {
		return DB().Delete(topic).Error
	}

	topic.Verification = model.VerificationCompleted
	topic.ChallangeId = 0
	topic.ChallangeSent = 0
	return DB().Save(topic).Error
}

func terminateTopic(topic *model.Topic) error {
	DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)
	topic.TopicId = 0

	if topic.IsBan || topic.Verification == model.VerificationNotCompleted {
		return DB().Save(topic).Error
	}

	return DB().Delete(topic).Error
}
