package bots

import (
	"Topicgram/i18n"

	botapi "github.com/OvyFlash/telegram-bot-api"
	formatter "gitlab.com/CoiaPrant/telegram-bot-formatter"
)

func (bot *BotAPI) sendSuccess(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Success()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *Bot) getWelcome(translator i18n.Translator, user_id int64) (string, []botapi.MessageEntity) {
	return translator.Welcome(false, "")
}

func (bot *Bot) sendWelcome(baseChat botapi.BaseChat, translator i18n.Translator, user_id int64) error {
	text, entities := bot.getWelcome(translator, user_id)
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendCaptcha(baseChat botapi.BaseChat, translator i18n.Translator, description *formatter.Builder, replyMarkup botapi.InlineKeyboardMarkup) error {
	baseChat.ReplyMarkup = replyMarkup

	text, entities := translator.Captcha(description)
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendCaptchaNotCompleted(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CaptchaNotCompleted()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendCaptchaFailed(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CaptchaFailed()
	messageConfig := botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	}

	_, err := bot.Send(messageConfig)
	return err
}

func (bot *BotAPI) sendCaptchaCompleted(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CaptchaCompleted()
	messageConfig := botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	}

	_, err := bot.Send(messageConfig)
	return err
}

func (bot *BotAPI) sendBanned(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Banned()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendBlocked(baseChat botapi.BaseChat, translator i18n.Translator, user_id int64) error {
	text, entities := translator.Blocked(user_id)
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendTerminated(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Terminated()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendSender(baseChat botapi.BaseChat, translator i18n.Translator, user *botapi.User) (botapi.Message, error) {
	text, entities := translator.Sender(user)
	return bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
}

func (bot *BotAPI) sendCaptchaNotify(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CaptchaNotify()
	messageConfig := botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	}

	_, err := bot.Send(messageConfig)
	return err
}

func (bot *BotAPI) sendCaptchaCompletedNotify(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CaptchaCompletedNotify()
	messageConfig := botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	}

	_, err := bot.Send(messageConfig)
	return err
}

func (bot *BotAPI) sendBanUser(baseChat botapi.BaseChat, translator i18n.Translator, user_id int64) error {
	text, entities := translator.BanUser(user_id)
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendUnbanUser(baseChat botapi.BaseChat, translator i18n.Translator, user_id int64) error {
	text, entities := translator.UnbanUser(user_id)
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}
