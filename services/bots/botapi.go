package bots

import (
	"encoding/json"
	"sync"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/clog"
)

type BotAPI struct {
	*botapi.BotAPI
	bot         sync.RWMutex
	topic       sync.Mutex
	mediaGroups mediaGroupCache
}

func NewBotAPI(b *botapi.BotAPI) *BotAPI {
	mediaGroups := NewMediaGroupCache()
	mediaGroups.AddAboutToDeleteItemCallback(func(item mediaGroupItem) {
		mediaGroup := item.Data()
		close(mediaGroup.done)
	})

	return &BotAPI{BotAPI: b, mediaGroups: mediaGroups}
}

func (bot *BotAPI) Request(c botapi.Chattable) (*botapi.APIResponse, error) {
	response, err := bot.BotAPI.Request(c)
	if err != nil {
		clog.Errorf("[Bot %d] request failed, error: %s", bot.Self.ID, err)
		return response, err
	}

	return response, err
}

func (bot *BotAPI) Send(c botapi.Chattable) (botapi.Message, error) {
	message, err := bot.BotAPI.Send(c)
	if err != nil {
		clog.Errorf("[Bot %d] send failed, error: %s", bot.Self.ID, err)
		return message, err
	}

	return message, err
}

func (bot *BotAPI) SendMediaGroup(config botapi.MediaGroupConfig) ([]botapi.Message, error) {
	messages, err := bot.BotAPI.SendMediaGroup(config)
	if err != nil {
		clog.Errorf("[Bot %d] sendMediaGroup failed, error: %s", bot.Self.ID, err)
		return messages, err
	}

	return messages, nil
}

// ForwardMessages forwards multi-messages and returns the resulting message ids.
func (bot *BotAPI) ForwardMessages(c botapi.ForwardMessagesConfig) ([]botapi.MessageID, error) {
	response, err := bot.BotAPI.Request(c)
	if err != nil {
		clog.Errorf("[Bot %d] forwardMessages failed, error: %s", bot.Self.ID, err)
		return nil, err
	}

	var messageIds []botapi.MessageID
	err = json.Unmarshal(response.Result, &messageIds)
	if err != nil {
		clog.Errorf("[Bot %d] forwardMessages failed, error: %s", bot.Self.ID, err)
		return nil, err
	}

	return messageIds, nil
}

func (bot *BotAPI) GetChatMember(config botapi.GetChatMemberConfig) (botapi.ChatMember, error) {
	members, err := bot.BotAPI.GetChatMember(config)
	if err == nil {
		return members, err
	}

	clog.Errorf("[Bot %d] getChatMember failed, error: %s", bot.Self.ID, err)
	return members, err
}
