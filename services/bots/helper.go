package bots

import (
	. "Topicgram/database"
	"Topicgram/model"
	"strings"

	botapi "github.com/OvyFlash/telegram-bot-api"
)

func isServiceMessage(msg *botapi.Message) bool {
	return (msg.NewChatTitle != "" ||
		msg.NewChatPhoto != nil || msg.DeleteChatPhoto ||
		msg.NewChatMembers != nil || msg.LeftChatMember != nil ||
		msg.GroupChatCreated || msg.SuperGroupChatCreated || msg.ChannelChatCreated ||
		msg.MessageAutoDeleteTimerChanged != nil ||
		msg.PinnedMessage != nil ||
		msg.BoostAdded != nil ||
		msg.ChatBackgroundSet != nil ||
		msg.ChecklistTasksDone != nil || msg.ChecklistTasksAdded != nil ||
		msg.DirectMessagePriceChanged != nil ||
		msg.ManagedBotCreated != nil ||
		msg.PaidMessagePriceChanged != nil ||
		msg.PollOptionAdded != nil || msg.PollOptionDeleted != nil ||
		msg.ForumTopicEdited != nil ||
		msg.GeneralForumTopicHidden != nil || msg.GeneralForumTopicUnhidden != nil ||
		msg.VideoChatScheduled != nil || msg.VideoChatStarted != nil || msg.VideoChatEnded != nil || msg.VideoChatParticipantsInvited != nil)
}

func isIgnoreMessage(msg *botapi.Message) bool {
	return (msg.ChatOwnerLeft != nil || msg.ChatOwnerChanged != nil ||
		msg.Invoice != nil || msg.SuccessfulPayment != nil || msg.RefundedPayment != nil ||
		msg.UsersShared != nil || msg.ChatShared != nil ||
		msg.Gift != nil || msg.UniqueGift != nil || msg.GiftUpgradeSent != nil ||
		msg.ConnectedWebsite != "" || msg.WriteAccessAllowed != nil ||
		msg.PassportData != nil ||
		msg.ProximityAlertTriggered != nil ||
		msg.GiveawayCreated != nil || msg.GiveawayWinners != nil || msg.GiveawayCompleted != nil ||
		msg.SuggestedPostApproved != nil || msg.SuggestedPostApprovalFailed != nil || msg.SuggestedPostDeclined != nil ||
		msg.SuggestedPostPaid != nil || msg.SuggestedPostRefunded != nil ||
		msg.WebAppData != nil || msg.ReplyMarkup != nil || msg.ForumTopicCreated != nil)
}

func isAllowedMessage(msg *botapi.Message) bool {
	return (msg.Text != "" || msg.RichMessage != nil ||
		msg.Animation != nil || msg.PremiumAnimation != nil ||
		msg.Audio != nil || msg.Document != nil ||
		msg.Photo != nil || msg.LivePhoto != nil ||
		msg.Sticker != nil ||
		msg.Video != nil || msg.VideoNote != nil ||
		msg.Voice != nil ||
		msg.Venue != nil || (msg.Location != nil && msg.Location.LivePeriod == 0))
}

func isAllowedEditMessage(msg *botapi.Message) bool {
	return (msg.Text != "" ||
		msg.Animation != nil || msg.PremiumAnimation != nil ||
		msg.Audio != nil || msg.Document != nil ||
		msg.Photo != nil || msg.LivePhoto != nil ||
		msg.Video != nil || msg.VideoNote != nil ||
		msg.Voice != nil)
}

func isBlocked(err *botapi.Error) bool {
	return strings.Contains(err.Message, "bot was blocked by the user")
}

func isThreadNotFound(err *botapi.Error) bool {
	return strings.Contains(err.Message, "message thread not found")
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
