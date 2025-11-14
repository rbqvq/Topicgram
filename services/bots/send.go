package bots

import (
	"Topicgram/i18n"

	botapi "github.com/OvyFlash/telegram-bot-api"
)

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

func (bot *BotAPI) sendSuccess(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Success()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
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
