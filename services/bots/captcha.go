package bots

import (
	"Topicgram/i18n"
	"Topicgram/model"
	"Topicgram/services/captcha"
	"time"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/clog"
	formatter "gitlab.com/CoiaPrant/telegram-bot-formatter"
	"gitlab.com/go-extension/rand"
)

const (
	CAPTCHA_DURATION = time.Minute

	CAPTCHA_OLDER_USER_ID = 1000000000
)

func (bot *Bot) shouldSendCaptcha(msg *botapi.Message) bool {
	// Premium bypass
	if msg.From.IsPremium {
		clog.Debugf("[Bot %d] user %d is premium, bypass the captcha", bot.Self.ID, msg.From.ID)
		return false
	}

	// Older user bypass
	if msg.From.ID <= CAPTCHA_OLDER_USER_ID {
		clog.Debugf("[Bot %d] user %d is older user, bypass the captcha", bot.Self.ID, msg.From.ID)
		return false
	}

	chat, _ := bot.GetChat(botapi.ChatInfoConfig{
		ChatConfig: msg.Chat.ChatConfig(),
	})

	// NFT user bypass
	if len(chat.ActiveUsernames) > 1 {
		clog.Debugf("[Bot %d] user %d assigns NFT username %v, bypass the captcha", bot.Self.ID, msg.From.ID, chat.ActiveUsernames)
		return false
	}

	return true
}

func (bot *Bot) captchaMath(translator i18n.Translator, topic *model.Topic) (*formatter.Builder, botapi.InlineKeyboardMarkup) {
	topic.ChallangeId = rand.Crypto.Uint64()

	problem, replyMarkup := captcha.NewMath(bot.Token, topic.ChallangeId)
	description := translator.CaptchaMath(CAPTCHA_DURATION, problem)
	return description, replyMarkup
}
