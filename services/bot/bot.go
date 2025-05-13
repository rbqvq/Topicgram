package bot

import (
	. "Topicgram/database"
	"Topicgram/i18n"
	"Topicgram/model"
	"Topicgram/utils"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/gin-gonic/gin"
	"gitlab.com/CoiaPrant/clog"
)

var (
	bot         *Bot
	secretToken string
)

func Init(botConfig *model.BotConfig) error {
	secretToken = utils.MD5(botConfig.WebHook.Host) + utils.SHA256(botConfig.Token)

	webhookConfig := botapi.WebhookConfig{
		URL: &url.URL{
			Scheme: "https",
			Host:   botConfig.WebHook.Host,
			Path:   "/topicgram/webhook",
		},
		MaxConnections: 100,
		SecretToken:    secretToken,
	}

	b, err := botapi.NewBotAPIWithClient(botConfig.Token, botapi.APIEndpoint, utils.BotClient)
	if err != nil {
		return err
	}

	{
		chatConfig := botapi.ChatConfig{
			ChatID: botConfig.GroupId,
		}

		chat, err := b.GetChat(botapi.ChatInfoConfig{
			ChatConfig: chatConfig,
		})
		if err != nil {
			return err
		}

		if !chat.IsForum {
			return fmt.Errorf("[Group %d] Topic mode required", botConfig.GroupId)
		}

		member, err := b.GetChatMember(botapi.GetChatMemberConfig{
			ChatConfigWithUser: botapi.ChatConfigWithUser{
				ChatConfig: chatConfig,
				UserID:     b.Self.ID,
			},
		})
		if err != nil {
			return err
		}

		if member.Status != "administrator" {
			return fmt.Errorf("[Group %d] Group administrator required", botConfig.GroupId)
		}

		if !member.CanDeleteMessages || !member.CanPinMessages || !member.CanManageTopics {
			return fmt.Errorf("[Group %d] Permissions (delete_messages, pin_messages, manage_topics) required", botConfig.GroupId)
		}
	}

	_, err = b.Request(webhookConfig)
	if err != nil {
		return err
	}

	i18n.Range(func(code string, translator i18n.Translator) {
		if code != "" && len(code) != 2 {
			return
		}

		b.Request(botapi.SetMyCommandsConfig{
			Commands: []botapi.BotCommand{
				{Command: "ban", Description: translator.CommandDescription_Ban()},
				{Command: "unban", Description: translator.CommandDescription_Unban()},
				{Command: "terminate", Description: translator.CommandDescription_Terminate()},
			},
			Scope: &botapi.BotCommandScope{
				Type:   "chat",
				ChatID: botConfig.GroupId,
			},
			LanguageCode: code,
		})
	})

	mediaGroups := NewMediaGroupCache()
	mediaGroups.AddAboutToDeleteItemCallback(func(item mediaGroupItem) {
		mediaGroup := item.Data()
		close(mediaGroup.done)
	})

	bot = &Bot{BotConfig: botConfig, BotAPI: &BotAPI{BotAPI: b, mediaGroups: mediaGroups}}
	clog.Success("[Bot] Load completed")
	return nil
}

func HookHandler(c *gin.Context) {
	token := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if token != secretToken {
		c.String(404, "404 page not found")
		return
	}

	update, err := bot.HandleUpdate(c.Request)
	if err != nil {
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go bot.handleUpdate(update)
	c.String(200, "OK")
}

type Bot struct {
	*model.BotConfig
	*BotAPI
}

func (bot *Bot) handleUpdate(update *botapi.Update) {
	defer func() {
		err := recover()
		if err != nil {
			clog.Errorf("panic recovered, error: %s", err)
		}
	}()

	chat := update.FromChat()
	if chat == nil {
		return
	}

	from := update.SentFrom()
	if from == nil {
		return
	}

	switch chat.ID {
	case bot.GroupId:
		if !chat.IsForum {
			text, entities := i18n.GetOrDefault(from.LanguageCode).Error_TopicRequired()
			bot.Send(botapi.MessageConfig{
				BaseChat: botapi.BaseChat{
					ChatConfig: botapi.ChatConfig{
						ChatID: chat.ID,
					},
				},
				Text:     text,
				Entities: entities,
			})
			return
		}

		switch {
		case update.EditedMessage != nil:
			bot.handleTopicEditMessage(update)
		case update.Message != nil:
			bot.handleTopicNewMessage(update)
		}
	default:
		if !chat.IsPrivate() {
			return
		}

		switch {
		case update.EditedMessage != nil:
			bot.handleUserEditMessage(update)
		case update.Message != nil:
			bot.handleUserNewMessage(update)
		}
	}
}

func generateMediaGroup(msgs []*botapi.Message, baseChat botapi.BaseChat) (botapi.MediaGroupConfig, error) {
	var medias []botapi.InputMedia
	for _, msg := range msgs {
		var inputMedia botapi.InputMedia
		baseInputMedia := botapi.BaseInputMedia{
			Caption:               msg.Caption,
			CaptionEntities:       msg.CaptionEntities,
			ShowCaptionAboveMedia: msg.ShowCaptionAboveMedia,
			HasSpoiler:            msg.HasMediaSpoiler,
		}

		switch {
		case msg.Audio != nil:
			media := msg.Audio

			baseInputMedia.Type = "audio"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaAudio{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
				Title:          media.Title,
				Performer:      media.Performer,
				Duration:       media.Duration,
			}

		case msg.Document != nil:
			media := msg.Document

			baseInputMedia.Type = "ducument"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaDocument{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
			}

		case msg.Photo != nil:
			var media botapi.PhotoSize
			for _, photo := range msg.Photo {
				if photo.FileSize > media.FileSize {
					media = photo
				}
			}

			baseInputMedia.Type = "photo"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			inputMedia = &botapi.InputMediaPhoto{
				BaseInputMedia: baseInputMedia,
			}

		case msg.Video != nil:
			media := msg.Video

			baseInputMedia.Type = "video"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaVideo{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
				Height:         media.Height,
				Width:          media.Width,
				Duration:       media.Duration,
			}
		default:
			return botapi.MediaGroupConfig{}, errors.New("unsupported media type in media group")
		}

		medias = append(medias, inputMedia)
	}

	return botapi.MediaGroupConfig{
		BaseChat: baseChat,
		Media:    medias,
	}, nil
}

func (bot *Bot) handleUserNewMessage(update *botapi.Update) {
	msg := update.Message
	translator := i18n.GetOrDefault(msg.From.LanguageCode)

	currentChatConfig := botapi.ChatConfig{
		ChatID: msg.Chat.ID,
	}
	currentChat := botapi.BaseChat{
		ChatConfig: currentChatConfig,
		ReplyParameters: botapi.ReplyParameters{
			MessageID: msg.MessageID,
		},
	}

	var mediaGroup *MediaGroup
	if msg.MediaGroupID != "" {
		_mediaGroup := &MediaGroup{}
		if bot.mediaGroups.NotFoundAdd(msg.MediaGroupID, mediaGroupLifeSpan, _mediaGroup) {
			done := make(chan struct{})
			_mediaGroup.done = done

			_mediaGroup.Add(msg)
			<-done
			_mediaGroup.Sort()
			mediaGroup = _mediaGroup
		} else {
			item, err := bot.mediaGroups.Value(msg.MediaGroupID)
			if err == nil {
				mediaGroup := item.Data()
				mediaGroup.Add(msg)
				return
			}
		}
	}

	// Message types
	switch {
	case msg.NewChatTitle != "", msg.NewChatPhoto != nil,
		msg.DeleteChatPhoto,
		msg.GroupChatCreated, msg.SuperGroupChatCreated, msg.ChannelChatCreated,
		msg.NewChatMembers != nil, msg.LeftChatMember != nil,
		msg.PinnedMessage != nil,
		msg.MessageAutoDeleteTimerChanged != nil,
		msg.ConnectedWebsite != "",
		msg.SuccessfulPayment != nil,
		msg.WriteAccessAllowed != nil,
		msg.ForumTopicReopened != nil, msg.ForumTopicClosed != nil, msg.ForumTopicEdited != nil,
		msg.GeneralForumTopicHidden != nil, msg.GeneralForumTopicUnhidden != nil,
		msg.VideoChatScheduled != nil, msg.VideoChatStarted != nil, msg.VideoChatEnded != nil, msg.VideoChatParticipantsInvited != nil:

		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: botapi.BaseChatMessage{
				ChatConfig: currentChatConfig,
				MessageID:  msg.MessageID,
			},
		})
		return
	case msg.WebAppData != nil, msg.ReplyMarkup != nil,
		msg.ForumTopicCreated != nil:
		return
	case msg.Text != "",
		msg.Animation != nil, msg.PremiumAnimation != nil,
		msg.Audio != nil, msg.Document != nil,
		msg.Photo != nil, msg.Sticker != nil,
		msg.Video != nil, msg.VideoNote != nil,
		msg.Voice != nil:

	default:
		text, entities := translator.Error_UnsupportedMessage()
		_, err := bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
			return
		}
		return
	}

	if strings.HasPrefix(msg.Text, "/") {
		command, _, _ := strings.Cut(msg.Text, " ")
		switch command {
		case "/start", "/help":
			text, entities := translator.Welcome(false, "")
			_, err := bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})

			if err != nil {
				clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
				return
			}
		default:
			text, entities := translator.Error_UnknwonCommand()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
		}
		return
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("user_id", msg.From.ID).Find(&topic).Error
	if err != nil {
		clog.Errorf("[DB] failed to query database, error: %s", err)

		text, entities := translator.Error_Database()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	botTranslator := i18n.GetOrDefault(bot.LanguageCode)
	botChatConfig := botapi.ChatConfig{
		ChatID: bot.GroupId,
	}
	botChat := botapi.BaseChat{
		ChatConfig: botChatConfig,
	}
	botTopic := botapi.BaseChat{
		ChatConfig: botChatConfig,
	}

	switch {
	case topic.IsBan:
		text, entities := translator.Banned()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	case topic.Id == 0:
		result, err := bot.Send(botapi.CreateForumTopicConfig{
			ChatConfig: botChatConfig,
			Name:       msg.From.FirstName + " " + msg.From.LastName,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to create topic in group, error: %s", err)

			{
				text, entities := botTranslator.Error_FailedToCreateTopic(false)
				bot.Send(botapi.MessageConfig{
					BaseChat: botChat,
					Text:     text,
					Entities: entities,
				})
			}

			{
				text, entities := translator.Error_FailedToCreateTopic(true)
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
			}
			return
		}

		topic.UserId = msg.From.ID
		topic.TopicId = result.MessageThreadID
		topic.LanguageCode = msg.From.LanguageCode
		DB().Create(&topic)
		botTopic.MessageThreadID = result.MessageThreadID

		{
			text, entities := botTranslator.Sender(msg.From)
			result, err = bot.Send(botapi.MessageConfig{
				BaseChat: botTopic,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
		}

		_, err = bot.Request(botapi.PinChatMessageConfig{
			BaseChatMessage: botapi.BaseChatMessage{
				ChatConfig: botChatConfig,
				MessageID:  result.MessageID,
			},
		})
		if err != nil {
			clog.Errorf("[Bot] failed to pin message in group, error: %s", err)
			return
		}
	default:
		botTopic.MessageThreadID = topic.TopicId
	}

	if msg.HasProtectedContent {
		text, entities := translator.Error_ForwardForbidden()
		_, err := bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
			return
		}
	}

	if msg.ForwardOrigin != nil {
		if mediaGroup == nil {
			result, err := bot.Send(botapi.ForwardConfig{
				BaseChat:  botTopic,
				FromChat:  currentChatConfig,
				MessageID: msg.MessageID,
			})
			if err != nil {
				err, ok := err.(*botapi.Error)
				if !ok {
					text, entities := translator.Error()
					bot.Send(botapi.MessageConfig{
						BaseChat: currentChat,
						Text:     text,
						Entities: entities,
					})
					clog.Errorf("[Bot] failed to forward message to group, error: %s", err)
					return
				}

				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     err.Message,
				})
				return
			}

			DB().Create(&model.Msg{
				TopicId:    topic.Id,
				UserMsgId:  msg.MessageID,
				TopicMsgId: result.MessageID,
			})
		} else {
			message_ids, err := bot.ForwardMessages(botapi.ForwardMessagesConfig{
				BaseChat:   botTopic,
				FromChat:   currentChatConfig,
				MessageIDs: mediaGroup.MessageIds(),
			})
			if err != nil {
				err, ok := err.(*botapi.Error)
				if !ok {
					text, entities := translator.Error()
					bot.Send(botapi.MessageConfig{
						BaseChat: currentChat,
						Text:     text,
						Entities: entities,
					})
					clog.Errorf("[Bot] failed to forward messages to group, error: %s", err)
					return
				}

				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     err.Message,
				})
				return
			}

			var msgs []model.Msg
			for i, msg := range mediaGroup.Messages {
				topic_message_id := message_ids[i].MessageID

				msgs = append(msgs, model.Msg{
					TopicId:    topic.Id,
					UserMsgId:  msg.MessageID,
					TopicMsgId: topic_message_id,
				})
			}

			DB().Create(msgs)
		}

		return
	}

	if msg.ReplyToMessage != nil {
		var message model.Msg
		err := DB().Where("topic_id", topic.Id).Where("user_msg_id", msg.ReplyToMessage.MessageID).Find(&message).Error
		if err != nil {
			clog.Errorf("[DB] failed to query database, error: %s", err)

			text, entities := translator.Error_Database()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			return
		}

		botTopic.ReplyParameters.MessageID = message.TopicMsgId

		if msg.Quote != nil && msg.Quote.IsManual {
			botTopic.ReplyParameters.Quote = msg.Quote.Text
			botTopic.ReplyParameters.QuoteEntities = msg.Quote.Entities
			botTopic.ReplyParameters.QuotePosition = msg.Quote.Position
		}
	}

	if mediaGroup != nil {
		mediaGroupConfig, err := generateMediaGroup(mediaGroup.Messages, botTopic)
		if err != nil {
			text, entities := translator.Error_UnsupportedMessage()
			_, err := bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
				return
			}
			return
		}

		results, err := bot.SendMediaGroup(mediaGroupConfig)
		if err != nil {
			err, ok := err.(*botapi.Error)
			if !ok {
				text, entities := translator.Error()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				clog.Errorf("[Bot] failed to send messages to group, error: %s", err)
				return
			}

			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     err.Message,
			})
			return
		}

		var msgs []model.Msg
		for i, msg := range mediaGroup.Messages {
			topic_message_id := results[i].MessageID

			msgs = append(msgs, model.Msg{
				TopicId:    topic.Id,
				UserMsgId:  msg.MessageID,
				TopicMsgId: topic_message_id,
			})
		}

		DB().Create(msgs)
		return
	}

	result, err := bot.Send(botapi.CopyMessageConfig{
		BaseChat:  botTopic,
		FromChat:  currentChatConfig,
		MessageID: msg.MessageID,
	})
	if err != nil {
		err, ok := err.(*botapi.Error)
		if !ok {
			text, entities := translator.Error()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			clog.Errorf("[Bot] failed to copy message to group, error: %s", err)
			return
		}

		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     err.Message,
		})
		return
	}

	DB().Create(&model.Msg{
		TopicId:    topic.Id,
		UserMsgId:  msg.MessageID,
		TopicMsgId: result.MessageID,
	})
}

func generateEditMessage(msg *botapi.Message, baseEdit botapi.BaseEdit) botapi.Chattable {
	var m botapi.Chattable
	switch {
	case msg.Text != "":
		m = botapi.EditMessageTextConfig{
			BaseEdit: baseEdit,
			Text:     msg.Text,
			Entities: msg.Entities,
		}
	case msg.Animation != nil,
		msg.PremiumAnimation != nil,
		msg.Audio != nil,
		msg.Document != nil,
		msg.Photo != nil,
		msg.Video != nil:

		var inputMedia botapi.InputMedia
		baseInputMedia := botapi.BaseInputMedia{
			Caption:               msg.Caption,
			CaptionEntities:       msg.CaptionEntities,
			ShowCaptionAboveMedia: msg.ShowCaptionAboveMedia,
			HasSpoiler:            msg.HasMediaSpoiler,
		}

		switch {
		case msg.Animation != nil:
			media := msg.Animation

			baseInputMedia.Type = "animation"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaAnimation{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
				Height:         media.Height,
				Width:          media.Width,
				Duration:       media.Duration,
			}

		case msg.PremiumAnimation != nil:
			media := msg.PremiumAnimation

			baseInputMedia.Type = "animation"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaAnimation{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
				Height:         media.Height,
				Width:          media.Width,
				Duration:       media.Duration,
			}

		case msg.Audio != nil:
			media := msg.Audio

			baseInputMedia.Type = "audio"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaAudio{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
				Title:          media.Title,
				Performer:      media.Performer,
				Duration:       media.Duration,
			}

		case msg.Document != nil:
			media := msg.Document

			baseInputMedia.Type = "ducument"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaDocument{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
			}

		case msg.Photo != nil:
			var media botapi.PhotoSize
			for _, photo := range msg.Photo {
				if photo.FileSize > media.FileSize {
					media = photo
				}
			}

			baseInputMedia.Type = "photo"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			inputMedia = &botapi.InputMediaPhoto{
				BaseInputMedia: baseInputMedia,
			}

		case msg.Video != nil:
			media := msg.Video

			baseInputMedia.Type = "video"
			baseInputMedia.Media = botapi.FileID(media.FileID)

			var thumb botapi.RequestFileData
			if media.Thumbnail != nil {
				thumb = botapi.FileID(media.Thumbnail.FileID)
			}

			inputMedia = &botapi.InputMediaVideo{
				BaseInputMedia: baseInputMedia,
				Thumb:          thumb,
				Height:         media.Height,
				Width:          media.Width,
				Duration:       media.Duration,
			}
		}

		m = botapi.EditMessageMediaConfig{
			BaseEdit: baseEdit,
			Media:    inputMedia,
		}
	case msg.Voice != nil,
		msg.VideoNote != nil:
		m = botapi.EditMessageCaptionConfig{
			BaseEdit:        baseEdit,
			Caption:         msg.Caption,
			CaptionEntities: msg.CaptionEntities,
		}
	}

	return m
}

func (bot *Bot) handleUserEditMessage(update *botapi.Update) {
	msg := update.EditedMessage
	translator := i18n.GetOrDefault(msg.From.LanguageCode)

	currentChatConfig := botapi.ChatConfig{
		ChatID: msg.Chat.ID,
	}
	currentChat := botapi.BaseChat{
		ChatConfig: currentChatConfig,
		ReplyParameters: botapi.ReplyParameters{
			MessageID: msg.MessageID,
		},
	}

	// Message types
	switch {
	case msg.NewChatTitle != "", msg.NewChatPhoto != nil,
		msg.DeleteChatPhoto,
		msg.GroupChatCreated, msg.SuperGroupChatCreated, msg.ChannelChatCreated,
		msg.NewChatMembers != nil, msg.LeftChatMember != nil,
		msg.PinnedMessage != nil,
		msg.MessageAutoDeleteTimerChanged != nil,
		msg.ConnectedWebsite != "",
		msg.SuccessfulPayment != nil,
		msg.WriteAccessAllowed != nil,
		msg.ForumTopicReopened != nil, msg.ForumTopicClosed != nil, msg.ForumTopicEdited != nil,
		msg.GeneralForumTopicHidden != nil, msg.GeneralForumTopicUnhidden != nil,
		msg.VideoChatScheduled != nil, msg.VideoChatStarted != nil, msg.VideoChatEnded != nil, msg.VideoChatParticipantsInvited != nil:

		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: botapi.BaseChatMessage{
				ChatConfig: currentChatConfig,
				MessageID:  msg.MessageID,
			},
		})
		return
	case msg.WebAppData != nil, msg.ReplyMarkup != nil,
		msg.ForumTopicCreated != nil:
		return
	case msg.Text != "",
		msg.Animation != nil, msg.PremiumAnimation != nil,
		msg.Audio != nil, msg.Document != nil,
		msg.Photo != nil,
		msg.Video != nil, msg.VideoNote != nil,
		msg.Voice != nil:

	default:
		text, entities := translator.Error_UnsupportedMessage()
		_, err := bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
			return
		}
		return
	}

	if bot.GroupId == 0 {
		text, entities := translator.Error_NoForwardGroup()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("user_id", msg.From.ID).Find(&topic).Error
	if err != nil {
		clog.Errorf("[DB] failed to query database, error: %s", err)

		text, entities := translator.Error_Database()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	switch {
	case topic.Id == 0:
		text, entities := translator.Error_FailedToEdit()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	case topic.IsBan:
		text, entities := translator.Banned()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	var message model.Msg
	err = DB().Where("topic_id", topic.Id).Where("user_msg_id", msg.MessageID).Find(&message).Error
	if err != nil {
		clog.Errorf("[DB] failed to query database, error: %s", err)

		text, entities := translator.Error_Database()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	if message.Id == 0 {
		text, entities := translator.Error_FailedToEdit()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	botEdit := botapi.BaseEdit{
		BaseChatMessage: botapi.BaseChatMessage{
			ChatConfig: botapi.ChatConfig{
				ChatID: bot.GroupId,
			},
			MessageID: message.TopicMsgId,
		},
	}

	m := generateEditMessage(msg, botEdit)

	_, err = bot.Send(m)
	if err != nil {
		err, ok := err.(*botapi.Error)
		if !ok {
			text, entities := translator.Error()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			clog.Errorf("[Bot] failed to edit message in group, error: %s", err)
			return
		}

		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     err.Message,
		})
		return
	}
}

func (bot *Bot) handleTopicNewMessage(update *botapi.Update) {
	msg := update.Message
	translator := i18n.GetOrDefault(msg.From.LanguageCode)

	currentChatConfig := botapi.ChatConfig{
		ChatID: msg.Chat.ID,
	}
	currentGroup := botapi.BaseChat{
		ChatConfig: currentChatConfig,
	}
	currentTopic := botapi.BaseChat{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
	}
	currentForum := botapi.BaseForum{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
	}
	currentChat := botapi.BaseChat{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
		ReplyParameters: botapi.ReplyParameters{
			MessageID: msg.MessageID,
		},
	}
	currentMessage := botapi.BaseChatMessage{
		ChatConfig: currentChatConfig,
		MessageID:  msg.MessageID,
	}

	// Message types
	var isUnsupportMessage bool
	switch {
	case msg.NewChatTitle != "", msg.NewChatPhoto != nil,
		msg.DeleteChatPhoto,
		msg.GroupChatCreated, msg.SuperGroupChatCreated, msg.ChannelChatCreated,
		msg.NewChatMembers != nil, msg.LeftChatMember != nil,
		msg.PinnedMessage != nil,
		msg.MessageAutoDeleteTimerChanged != nil,
		msg.ConnectedWebsite != "",
		msg.SuccessfulPayment != nil,
		msg.WriteAccessAllowed != nil,
		msg.ForumTopicEdited != nil,
		msg.GeneralForumTopicHidden != nil, msg.GeneralForumTopicUnhidden != nil,
		msg.VideoChatScheduled != nil, msg.VideoChatStarted != nil, msg.VideoChatEnded != nil, msg.VideoChatParticipantsInvited != nil:

		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		return
	case msg.WebAppData != nil, msg.ReplyMarkup != nil,
		msg.ForumTopicCreated != nil:
		return
	case msg.MigrateFromChatID != 0:
		bot.bot.Lock()
		defer bot.bot.Unlock()

		if msg.MigrateFromChatID == bot.GroupId {
			bot.GroupId = msg.Chat.ID
			clog.Infof("[Bot] Group migrated to %d, please update config", msg.Chat.ID)
		}
		return
	case msg.MigrateToChatID != 0:
		bot.bot.Lock()
		defer bot.bot.Unlock()

		if msg.Chat.ID == bot.GroupId {
			bot.GroupId = msg.MigrateToChatID
			clog.Infof("[Bot] Group migrated to %d, please update config", msg.MigrateToChatID)
		}
		return
	case msg.ForumTopicClosed != nil:
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})

		// Ignored self message
		if msg.From.ID == bot.Self.ID {
			return
		}

		// General Topic
		if msg.MessageThreadID == 0 {
			return
		}

		bot.bot.RLock()
		defer bot.bot.RUnlock()

		bot.topic.Lock()
		defer bot.topic.Unlock()

		var topic model.Topic
		err := DB().Where("topic_id", msg.MessageThreadID).Find(&topic).Error
		if err != nil {
			clog.Errorf("[DB] failed to query database, error: %s", err)

			text, entities := translator.Error_Database()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentTopic,
				Text:     text,
				Entities: entities,
			})
			return
		}

		if topic.Id == 0 {
			return
		}

		if topic.IsBan {
			return
		}

		topic.IsBan = true
		err = DB().Save(&topic).Error
		if err != nil {
			clog.Errorf("[DB] failed to update database, error: %s", err)

			text, entities := translator.Error_Database()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentTopic,
				Text:     text,
				Entities: entities,
			})
			return
		}

		userTranslator := i18n.GetOrDefault(topic.LanguageCode)
		userChat := botapi.BaseChat{
			ChatConfig: botapi.ChatConfig{
				ChatID: topic.UserId,
			},
		}

		{
			text, entities := userTranslator.Banned()
			_, err = bot.Send(botapi.MessageConfig{
				BaseChat: userChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in user chat, error: %s", err)
				return
			}
		}

		{
			text, entities := translator.BanUser(topic.UserId)
			_, err = bot.Send(botapi.MessageConfig{
				BaseChat: currentTopic,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
		}
		return

	case msg.ForumTopicReopened != nil:
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})

		// Ignored self message
		if msg.From.ID == bot.Self.ID {
			return
		}

		// General Topic
		if msg.MessageThreadID == 0 {
			return
		}

		bot.bot.RLock()
		defer bot.bot.RUnlock()

		bot.topic.Lock()
		defer bot.topic.Unlock()

		var topic model.Topic
		err := DB().Where("topic_id", msg.MessageThreadID).Find(&topic).Error
		if err != nil {
			clog.Errorf("[DB] failed to query database, error: %s", err)

			text, entities := translator.Error_Database()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentTopic,
				Text:     text,
				Entities: entities,
			})
			return
		}

		if topic.Id == 0 {
			return
		}

		if !topic.IsBan {
			return
		}

		topic.IsBan = false

		err = DB().Save(&topic).Error
		if err != nil {
			clog.Errorf("[DB] failed to update database, error: %s", err)

			text, entities := translator.Error_Database()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentTopic,
				Text:     text,
				Entities: entities,
			})
			return
		}

		text, entities := translator.UnbanUser(topic.UserId)
		_, err = bot.Send(botapi.MessageConfig{
			BaseChat: currentTopic,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in group, error: %s", err)
			return
		}
		return
	case msg.Text != "",
		msg.Animation != nil, msg.PremiumAnimation != nil,
		msg.Audio != nil, msg.Document != nil,
		msg.Photo != nil, msg.Sticker != nil,
		msg.Video != nil, msg.VideoNote != nil,
		msg.Voice != nil:

		// Ignored self message
		if msg.From.ID == bot.Self.ID {
			return
		}

	default:
		// Ignored self message
		if msg.From.ID == bot.Self.ID {
			return
		}

		isUnsupportMessage = true
	}

	// General Topic
	if msg.MessageThreadID == 0 {
		if !strings.HasPrefix(msg.Text, "/") {
			return
		}

		command, args, _ := strings.Cut(msg.Text, " ")
		switch command {
		case "/ban", "/ban@" + bot.Self.UserName:
			user_id, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				text, entities := translator.CommandUsage_Ban()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			bot.bot.RLock()
			defer bot.bot.RUnlock()

			bot.topic.Lock()
			defer bot.topic.Unlock()

			var topic model.Topic
			err = DB().Where("user_id", user_id).Find(&topic).Error
			if err != nil {
				clog.Errorf("[DB] failed to query database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			if topic.IsBan {
				text, entities := translator.BanUser(user_id)
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			topic.IsBan = true

			if topic.Id == 0 {
				topic.UserId = user_id
				DB().Create(&topic)
			} else {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})

				topic.TopicId = 0
				err = DB().Save(&topic).Error
				if err != nil {
					clog.Errorf("[DB] failed to update database, error: %s", err)

					text, entities := translator.Error_Database()
					bot.Send(botapi.MessageConfig{
						BaseChat: currentChat,
						Text:     text,
						Entities: entities,
					})
					return
				}

				DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)
			}

			text, entities := translator.BanUser(user_id)
			_, err = bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
			return

		case "/unban", "/unban@" + bot.Self.UserName:
			user_id, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				text, entities := translator.CommandUsage_Unban()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			bot.bot.RLock()
			defer bot.bot.RUnlock()

			bot.topic.Lock()
			defer bot.topic.Unlock()

			var topic model.Topic
			err = DB().Where("user_id", user_id).Find(&topic).Error
			if err != nil {
				clog.Errorf("[DB] failed to query database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			if topic.Id == 0 {
				text, entities := translator.UnbanUser(user_id)
				_, err = bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in group, error: %s", err)
					return
				}
			}

			topic.IsBan = false

			if topic.TopicId == 0 {
				err = DB().Delete(&topic).Error
			} else {
				bot.Request(botapi.ReopenForumTopicConfig{
					BaseForum: currentForum,
				})

				err = DB().Save(&topic).Error
			}

			if err != nil {
				clog.Errorf("[DB] failed to update database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			text, entities := translator.UnbanUser(user_id)
			_, err = bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
			return

		case "/terminate", "/terminate@" + bot.Self.UserName:
			user_id, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				text, entities := translator.CommandUsage_Terminate()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			bot.bot.RLock()
			defer bot.bot.RUnlock()

			bot.topic.Lock()
			defer bot.topic.Unlock()

			var topic model.Topic
			err = DB().Where("user_id", user_id).Find(&topic).Error
			if err != nil {
				clog.Errorf("[DB] failed to query database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			if topic.Id == 0 {
				text, entities := translator.Success()
				_, err = bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in group, error: %s", err)
					return
				}
				return
			}

			if topic.TopicId != 0 {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})

				topic.TopicId = 0
			}

			if topic.IsBan {
				err = DB().Save(&topic).Error
			} else {
				err = DB().Delete(&topic).Error
			}

			if err != nil {
				clog.Errorf("[DB] failed to update database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			text, entities := translator.Success()
			_, err = bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
			return

		default:
			if strings.HasSuffix(command, "@"+bot.Self.UserName) {
				text, entities := translator.Error_UnknwonCommand()
				_, err := bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
					return
				}
				return
			}
		}
		return
	}

	var mediaGroup *MediaGroup
	if msg.MediaGroupID != "" {
		_mediaGroup := &MediaGroup{}
		if bot.mediaGroups.NotFoundAdd(msg.MediaGroupID, mediaGroupLifeSpan, _mediaGroup) {
			done := make(chan struct{})
			_mediaGroup.done = done

			_mediaGroup.Add(msg)
			<-done
			_mediaGroup.Sort()
			mediaGroup = _mediaGroup
		} else {
			item, err := bot.mediaGroups.Value(msg.MediaGroupID)
			if err == nil {
				mediaGroup := item.Data()
				mediaGroup.Add(msg)
				return
			}
		}
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("topic_id", msg.MessageThreadID).Find(&topic).Error
	if err != nil {
		clog.Errorf("[DB] failed to query database, error: %s", err)

		text, entities := translator.Error_Database()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	if topic.Id == 0 {
		return
	}

	if isUnsupportMessage {
		text, entities := translator.Error_UnsupportedMessage()
		_, err := bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in group, error: %s", err)
			return
		}
		return
	}

	userTranslator := i18n.GetOrDefault(topic.LanguageCode)
	userChat := botapi.BaseChat{
		ChatConfig: botapi.ChatConfig{
			ChatID: topic.UserId,
		},
	}

	if strings.HasPrefix(msg.Text, "/") {
		command, _, _ := strings.Cut(msg.Text, " ")
		switch command {
		case "/ban", "/ban@" + bot.Self.UserName:
			bot.Request(botapi.DeleteForumTopicConfig{
				BaseForum: currentForum,
			})

			if topic.IsBan {
				text, entities := translator.BanUser(topic.UserId)
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			topic.TopicId = 0
			topic.IsBan = true

			err = DB().Save(&topic).Error
			if err != nil {
				clog.Errorf("[DB] failed to update database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			{
				text, entities := userTranslator.Banned()
				_, err := bot.Send(botapi.MessageConfig{
					BaseChat: userChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in group, error: %s", err)
					return
				}
			}

			{
				text, entities := translator.BanUser(topic.UserId)
				_, err = bot.Send(botapi.MessageConfig{
					BaseChat: currentGroup,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in group, error: %s", err)
					return
				}
			}
			return

		case "/unban", "/unban@" + bot.Self.UserName:
			if !topic.IsBan {
				text, entities := translator.UnbanUser(topic.UserId)
				_, err = bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in group, error: %s", err)
					return
				}
				return
			}

			bot.Request(botapi.ReopenForumTopicConfig{
				BaseForum: currentForum,
			})

			topic.IsBan = false
			err = DB().Save(&topic).Error

			if err != nil {
				clog.Errorf("[DB] failed to update database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			text, entities := translator.UnbanUser(topic.UserId)
			_, err = bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
			return

		case "/terminate", "/terminate@" + bot.Self.UserName:
			bot.Request(botapi.DeleteForumTopicConfig{
				BaseForum: currentForum,
			})
			topic.TopicId = 0

			if topic.IsBan {
				err = DB().Save(&topic).Error
			} else {
				err = DB().Delete(&topic).Error
			}

			if err != nil {
				clog.Errorf("[DB] failed to update database, error: %s", err)

				text, entities := translator.Error_Database()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				return
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			if !topic.IsBan {
				text, entities := userTranslator.Terminated()
				_, err := bot.Send(botapi.MessageConfig{
					BaseChat: userChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in group, error: %s", err)
					return
				}
			}
			return

		default:
			if strings.HasSuffix(command, "@"+bot.Self.UserName) {
				text, entities := translator.Error_UnknwonCommand()
				_, err := bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				if err != nil {
					clog.Errorf("[Bot] failed to send message in chat, error: %s", err)
					return
				}
				return
			}
		}
	}

	if msg.HasProtectedContent {
		text, entities := translator.Error_ForwardForbidden()
		_, err := bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in group, error: %s", err)
			return
		}
	}

	if msg.ForwardOrigin != nil {
		if mediaGroup == nil {
			result, err := bot.Send(botapi.ForwardConfig{
				BaseChat:  userChat,
				FromChat:  currentChatConfig,
				MessageID: msg.MessageID,
			})
			if err != nil {
				err, ok := err.(*botapi.Error)
				if !ok {
					text, entities := translator.Error()
					bot.Send(botapi.MessageConfig{
						BaseChat: currentChat,
						Text:     text,
						Entities: entities,
					})
					clog.Errorf("[Bot] failed to forward message to user chat, error: %s", err)
					return
				}

				if strings.Contains(err.Message, "bot was blocked by the user") {
					bot.Request(botapi.DeleteForumTopicConfig{
						BaseForum: currentForum,
					})
					topic.TopicId = 0

					if topic.IsBan {
						DB().Save(&topic)
					} else {
						DB().Delete(&topic)
					}

					DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

					text, entities := translator.Blocked(topic.UserId)
					bot.Send(botapi.MessageConfig{
						BaseChat: currentGroup,
						Text:     text,
						Entities: entities,
					})
					return
				}

				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     err.Message,
				})
				return
			}

			DB().Create(&model.Msg{
				TopicId:    topic.Id,
				UserMsgId:  result.MessageID,
				TopicMsgId: msg.MessageID,
			})
		} else {
			message_ids, err := bot.ForwardMessages(botapi.ForwardMessagesConfig{
				BaseChat:   userChat,
				FromChat:   currentChatConfig,
				MessageIDs: mediaGroup.MessageIds(),
			})
			if err != nil {
				err, ok := err.(*botapi.Error)
				if !ok {
					text, entities := translator.Error()
					bot.Send(botapi.MessageConfig{
						BaseChat: currentChat,
						Text:     text,
						Entities: entities,
					})
					clog.Errorf("[Bot] failed to forward messages to user chat, error: %s", err)
					return
				}

				if strings.Contains(err.Message, "bot was blocked by the user") {
					bot.Request(botapi.DeleteForumTopicConfig{
						BaseForum: currentForum,
					})
					topic.TopicId = 0

					if topic.IsBan {
						DB().Save(&topic)
					} else {
						DB().Delete(&topic)
					}

					DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

					text, entities := translator.Blocked(topic.UserId)
					bot.Send(botapi.MessageConfig{
						BaseChat: currentGroup,
						Text:     text,
						Entities: entities,
					})
					return
				}

				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     err.Message,
				})
				return
			}

			var msgs []model.Msg
			for i, msg := range mediaGroup.Messages {
				user_message_id := message_ids[i].MessageID

				msgs = append(msgs, model.Msg{
					TopicId:    topic.Id,
					UserMsgId:  user_message_id,
					TopicMsgId: msg.MessageID,
				})
			}

			DB().Create(msgs)
		}

		return
	}

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.MessageID != msg.MessageThreadID {
		var message model.Msg
		err := DB().Where("topic_id", topic.Id).Where("topic_msg_id", msg.ReplyToMessage.MessageID).Find(&message).Error
		if err != nil {
			clog.Errorf("[DB] failed to query database, error: %s", err)

			text, entities := translator.Error_Database()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			return
		}

		userChat.ReplyParameters.MessageID = message.UserMsgId

		if msg.Quote != nil && msg.Quote.IsManual {
			userChat.ReplyParameters.Quote = msg.Quote.Text
			userChat.ReplyParameters.QuoteEntities = msg.Quote.Entities
			userChat.ReplyParameters.QuotePosition = msg.Quote.Position
		}
	}

	if mediaGroup != nil {
		mediaGroupConfig, err := generateMediaGroup(mediaGroup.Messages, userChat)
		if err != nil {
			text, entities := translator.Error_UnsupportedMessage()
			_, err := bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			if err != nil {
				clog.Errorf("[Bot] failed to send message in group, error: %s", err)
				return
			}
			return
		}

		results, err := bot.SendMediaGroup(mediaGroupConfig)
		if err != nil {
			err, ok := err.(*botapi.Error)
			if !ok {
				text, entities := translator.Error()
				bot.Send(botapi.MessageConfig{
					BaseChat: currentChat,
					Text:     text,
					Entities: entities,
				})
				clog.Errorf("[Bot] failed to send messages to user chat, error: %s", err)
				return
			}

			if strings.Contains(err.Message, "bot was blocked by the user") {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})
				topic.TopicId = 0

				if topic.IsBan {
					DB().Save(&topic)
				} else {
					DB().Delete(&topic)
				}

				DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

				text, entities := translator.Blocked(topic.UserId)
				bot.Send(botapi.MessageConfig{
					BaseChat: currentGroup,
					Text:     text,
					Entities: entities,
				})
				return
			}

			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     err.Message,
			})
			return
		}

		var msgs []model.Msg
		for i, msg := range mediaGroup.Messages {
			user_message_id := results[i].MessageID

			msgs = append(msgs, model.Msg{
				TopicId:    topic.Id,
				UserMsgId:  user_message_id,
				TopicMsgId: msg.MessageID,
			})
		}

		DB().Create(msgs)
		return
	}

	result, err := bot.Send(botapi.CopyMessageConfig{
		BaseChat:  userChat,
		FromChat:  currentChatConfig,
		MessageID: msg.MessageID,
	})
	if err != nil {
		err, ok := err.(*botapi.Error)
		if !ok {
			text, entities := translator.Error()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			clog.Errorf("[Bot] failed to send message in group, error: %s", err)
			return
		}

		if strings.Contains(err.Message, "bot was blocked by the user") {
			bot.Request(botapi.DeleteForumTopicConfig{
				BaseForum: currentForum,
			})
			topic.TopicId = 0

			if topic.IsBan {
				DB().Save(&topic)
			} else {
				DB().Delete(&topic)
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			text, entities := translator.Blocked(topic.UserId)
			bot.Send(botapi.MessageConfig{
				BaseChat: currentGroup,
				Text:     text,
				Entities: entities,
			})
			return
		}

		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     err.Message,
		})
		return
	}

	DB().Create(&model.Msg{
		TopicId:    topic.Id,
		UserMsgId:  result.MessageID,
		TopicMsgId: msg.MessageID,
	})
}

func (bot *Bot) handleTopicEditMessage(update *botapi.Update) {
	msg := update.EditedMessage

	// Ignore General Topic
	if msg.MessageThreadID == 0 {
		return
	}

	translator := i18n.GetOrDefault(msg.From.LanguageCode)

	currentChatConfig := botapi.ChatConfig{
		ChatID: msg.Chat.ID,
	}
	currentGroup := botapi.BaseChat{
		ChatConfig: currentChatConfig,
	}
	currentForum := botapi.BaseForum{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
	}
	currentChat := botapi.BaseChat{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
		ReplyParameters: botapi.ReplyParameters{
			MessageID: msg.MessageID,
		},
	}
	currentMessage := botapi.BaseChatMessage{
		ChatConfig: currentChatConfig,
		MessageID:  msg.MessageID,
	}

	// Message types
	var isUnsupportMessage bool
	switch {
	case msg.NewChatTitle != "", msg.NewChatPhoto != nil,
		msg.DeleteChatPhoto,
		msg.GroupChatCreated, msg.SuperGroupChatCreated, msg.ChannelChatCreated,
		msg.NewChatMembers != nil, msg.LeftChatMember != nil,
		msg.PinnedMessage != nil,
		msg.MessageAutoDeleteTimerChanged != nil,
		msg.ConnectedWebsite != "",
		msg.SuccessfulPayment != nil,
		msg.WriteAccessAllowed != nil,
		msg.ForumTopicReopened != nil, msg.ForumTopicClosed != nil, msg.ForumTopicEdited != nil,
		msg.GeneralForumTopicHidden != nil, msg.GeneralForumTopicUnhidden != nil,
		msg.VideoChatScheduled != nil, msg.VideoChatStarted != nil, msg.VideoChatEnded != nil, msg.VideoChatParticipantsInvited != nil:

		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		return

	case msg.WebAppData != nil, msg.ReplyMarkup != nil,
		msg.ForumTopicCreated != nil:
		return

	case msg.Text != "",
		msg.Animation != nil, msg.PremiumAnimation != nil,
		msg.Audio != nil, msg.Document != nil,
		msg.Photo != nil,
		msg.Video != nil, msg.VideoNote != nil,
		msg.Voice != nil:

		// Ignored self message
		if msg.From.ID == bot.Self.ID {
			return
		}

	default:
		// Ignored self message
		if msg.From.ID == bot.Self.ID {
			return
		}

		isUnsupportMessage = true
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("topic_id", msg.MessageThreadID).Find(&topic).Error
	if err != nil {
		clog.Errorf("[DB] failed to query database, error: %s", err)

		text, entities := translator.Error_Database()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	if topic.Id == 0 {
		return
	}

	if isUnsupportMessage {
		text, entities := translator.Error_UnsupportedMessage()
		_, err := bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		if err != nil {
			clog.Errorf("[Bot] failed to send message in group, error: %s", err)
			return
		}
		return
	}

	var message model.Msg
	err = DB().Where("topic_id", topic.Id).Where("topic_msg_id", msg.MessageID).Find(&message).Error
	if err != nil {
		clog.Errorf("[DB] failed to query database, error: %s", err)

		text, entities := translator.Error_Database()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	if message.Id == 0 {
		text, entities := translator.Error_FailedToEdit()
		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     text,
			Entities: entities,
		})
		return
	}

	userEdit := botapi.BaseEdit{
		BaseChatMessage: botapi.BaseChatMessage{
			ChatConfig: botapi.ChatConfig{
				ChatID: topic.UserId,
			},
			MessageID: message.UserMsgId,
		},
	}

	m := generateEditMessage(msg, userEdit)

	_, err = bot.Send(m)
	if err != nil {
		err, ok := err.(*botapi.Error)
		if !ok {
			text, entities := translator.Error()
			bot.Send(botapi.MessageConfig{
				BaseChat: currentChat,
				Text:     text,
				Entities: entities,
			})
			clog.Errorf("[Bot] failed to edit message in user chat, error: %s", err)
			return
		}

		if strings.Contains(err.Message, "bot was blocked by the user") {
			bot.Request(botapi.DeleteForumTopicConfig{
				BaseForum: currentForum,
			})
			topic.TopicId = 0

			if topic.IsBan {
				DB().Save(&topic)
			} else {
				DB().Delete(&topic)
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			text, entities := translator.Blocked(topic.UserId)
			bot.Send(botapi.MessageConfig{
				BaseChat: currentGroup,
				Text:     text,
				Entities: entities,
			})
			return
		}

		bot.Send(botapi.MessageConfig{
			BaseChat: currentChat,
			Text:     err.Message,
		})
		return
	}
}
