package bots

import (
	"Topicgram/i18n"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/clog"
)

func (bot *BotAPI) sendError(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendTelegramError(baseChat botapi.BaseChat, e *botapi.Error) error {
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     e.Message,
	})
	return err
}

func (bot *BotAPI) sendDatabaseError(baseChat botapi.BaseChat, translator i18n.Translator, err error) error {
	clog.Errorf("[DB] execute error: %s", err)

	text, entities := translator.Error_Database()
	_, err = bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendUnsupportedMessage(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error_UnsupportedMessage()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendUnknownCommand(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error_UnknwonCommand()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendTopicRequired(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error_TopicRequired()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendNoForwardGroup(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error_NoForwardGroup()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendFailedToCreateTopic(baseChat botapi.BaseChat, translator i18n.Translator, forUser bool) error {
	text, entities := translator.Error_FailedToCreateTopic(forUser)
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendForwardForbidden(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error_ForwardForbidden()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendFailedToEdit(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.Error_FailedToEdit()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}
