package bots

import (
	"Topicgram/i18n"

	botapi "github.com/OvyFlash/telegram-bot-api"
)

func (bot *BotAPI) sendCommandUsageBan(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CommandUsage_Ban()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendCommandUsageUnban(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CommandUsage_Unban()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}

func (bot *BotAPI) sendCommandUsageTerminate(baseChat botapi.BaseChat, translator i18n.Translator) error {
	text, entities := translator.CommandUsage_Terminate()
	_, err := bot.Send(botapi.MessageConfig{
		BaseChat: baseChat,
		Text:     text,
		Entities: entities,
	})
	return err
}
