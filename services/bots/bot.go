package bots

import (
	. "Topicgram/database"
	"Topicgram/i18n"
	"Topicgram/model"
	"Topicgram/services/captcha"
	"strconv"
	"strings"
	"time"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/clog"
)

type Bot struct {
	*model.BotConfig
	*BotAPI
}

func Recover() {
	err := recover()
	if err != nil {
		clog.Errorf("panic recovered, error: %s", err)
		return
	}
}

func (bot *Bot) handleUpdate(update *botapi.Update) {
	defer Recover()

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
			currentChat := botapi.BaseChat{
				ChatConfig: botapi.ChatConfig{
					ChatID: chat.ID,
				},
			}
			translator := i18n.GetOrDefault(from.LanguageCode)

			bot.sendTopicRequired(currentChat, translator)
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
		case update.CallbackQuery != nil:
			bot.handleUserVerification(update)
		}
	}
}

func (bot *Bot) handleUserVerification(update *botapi.Update) {
	callback := update.CallbackQuery
	if len(callback.Data) == 0 {
		return
	}

	msg := callback.Message
	if msg == nil {
		return
	}

	if bot.GroupId == 0 {
		return
	}

	currentChatConfig := botapi.ChatConfig{
		ChatID: msg.Chat.ID,
	}
	currentChat := botapi.BaseChat{
		ChatConfig: currentChatConfig,
		ReplyParameters: botapi.ReplyParameters{
			AllowSendingWithoutReply: true,
			MessageID:                msg.MessageID,
		},
	}
	currentMessage := botapi.BaseChatMessage{
		ChatConfig: currentChatConfig,
		MessageID:  msg.MessageID,
	}
	translator := i18n.GetOrDefault(callback.From.LanguageCode)

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("user_id", callback.From.ID).Find(&topic).Error
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	if topic.Id == 0 || topic.Verification != model.VerificationNotCompleted || topic.ChallangeId == 0 || topic.ChallangeSent == 0 {
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		return
	}

	if topic.IsBan {
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		bot.sendBanned(currentChat, translator)
		return
	}

	challangeId := topic.ChallangeId
	notAfter := time.Unix(topic.ChallangeSent, 0).Add(CAPTCHA_DURATION)

	topic.ChallangeSent = 0
	topic.ChallangeId = 0
	err = saveTopic(&topic)
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	if time.Now().After(notAfter) {
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		bot.sendCaptchaFailed(currentChat, translator)
		return
	}

	if !captcha.CheckMath(bot.Token, challangeId, callback.Data) {
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		bot.sendCaptchaFailed(currentChat, translator)
		return
	}

	topic.Verification = model.VerificationCompleted
	err = saveTopic(&topic)
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	bot.Request(botapi.DeleteMessageConfig{
		BaseChatMessage: currentMessage,
	})
	bot.sendCaptchaCompleted(currentChat, translator)

	if topic.TopicId == 0 {
		return
	}

	botTranslator := i18n.GetOrDefault(bot.LanguageCode)
	botChatConfig := botapi.ChatConfig{
		ChatID: bot.GroupId,
	}
	botTopic := botapi.BaseChat{
		ChatConfig:      botChatConfig,
		MessageThreadID: topic.TopicId,
	}
	bot.sendCaptchaCompletedNotify(botTopic, botTranslator)
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

	mediaGroup, skip := bot.getMediaGroup(msg)
	if skip {
		return
	}

	switch {
	case isServiceMessage(msg):
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: botapi.BaseChatMessage{
				ChatConfig: currentChatConfig,
				MessageID:  msg.MessageID,
			},
		})
		return
	case isIgnoreMessage(msg):
		return
	case isAllowedMessage(msg):
	default:
		bot.sendUnsupportedMessage(currentChat, translator)
		return
	}

	if strings.HasPrefix(msg.Text, "/") {
		command, _, _ := strings.Cut(msg.Text, " ")
		switch command {
		case "/start", "/help":
			bot.sendWelcome(currentChat, translator, msg.From.ID)
			return
		}
	}

	if bot.GroupId == 0 {
		bot.sendNoForwardGroup(currentChat, translator)
		return
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("user_id", msg.From.ID).Find(&topic).Error
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
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
	botTopic.MessageThreadID = topic.TopicId

	switch topic.Verification {
	case model.VerificationNotSent:
		if bot.shouldSendCaptcha(msg) {
			description, replyMarkup := bot.captchaMath(translator, &topic)
			err := bot.sendCaptcha(currentChat, translator, description, replyMarkup)
			if err != nil {
				return
			}

			topic.UserId = msg.From.ID
			topic.Verification = model.VerificationNotCompleted
			topic.ChallangeSent = time.Now().Unix()
			topic.LanguageCode = msg.From.LanguageCode

			err = saveTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			if topic.TopicId != 0 {
				bot.sendCaptchaNotify(botTopic, botTranslator)
			}
			return
		}
	case model.VerificationNotCompleted:
		bot.sendCaptchaNotCompleted(currentChat, translator)
		return
	}

	switch {
	case topic.IsBan:
		bot.sendBanned(currentChat, translator)
		return
	case topic.Id == 0:
		topic.UserId = msg.From.ID
		topic.LanguageCode = msg.From.LanguageCode
		fallthrough
	case topic.TopicId == 0:
		createdTopic, err := bot.Send(botapi.CreateForumTopicConfig{
			ChatConfig: botChatConfig,
			Name:       msg.From.FirstName + " " + msg.From.LastName,
		})
		if err != nil {
			bot.sendFailedToCreateTopic(botChat, botTranslator, false)
			bot.sendFailedToCreateTopic(currentChat, translator, true)
			return
		}

		botTopic.MessageThreadID = createdTopic.MessageThreadID
		topic.TopicId = createdTopic.MessageThreadID
		saveTopic(&topic)

		message, err := bot.sendSender(botTopic, botTranslator, msg.From)
		if err != nil {
			return
		}

		bot.Request(botapi.PinChatMessageConfig{
			BaseChatMessage: botapi.BaseChatMessage{
				ChatConfig: botChatConfig,
				MessageID:  message.MessageID,
			},
		})
	}

	if msg.HasProtectedContent {
		bot.sendForwardForbidden(currentChat, translator)
		return
	}

	sendError := func(err error) {
		if err, ok := err.(*botapi.Error); ok {
			bot.sendTelegramError(currentChat, err)
			return
		}

		bot.sendError(currentChat, translator)
	}

	if msg.ForwardOrigin != nil {
		if mediaGroup != nil {
			messageIds, err := bot.ForwardMessages(botapi.ForwardMessagesConfig{
				BaseChat:   botTopic,
				FromChat:   currentChatConfig,
				MessageIDs: mediaGroup.MessageIds(),
			})
			if err != nil {
				sendError(err)
				return
			}

			if len(mediaGroup.Messages) != len(messageIds) {
				bot.sendError(currentChat, translator)
				clog.Errorf("[Bot %d] messages length mismatch, want: %d, got: %d", len(mediaGroup.Messages), len(messageIds))
				return
			}

			msgs := make([]model.Msg, 0, len(mediaGroup.Messages))
			for i, msg := range mediaGroup.Messages {
				topic_message_id := messageIds[i].MessageID

				msgs = append(msgs, model.Msg{
					TopicId:    topic.Id,
					UserMsgId:  msg.MessageID,
					TopicMsgId: topic_message_id,
				})
			}

			DB().Create(msgs)
			return
		}

		message, err := bot.Send(botapi.ForwardConfig{
			BaseChat:  botTopic,
			FromChat:  currentChatConfig,
			MessageID: msg.MessageID,
		})
		if err != nil {
			sendError(err)
			return
		}

		DB().Create(&model.Msg{
			TopicId:    topic.Id,
			UserMsgId:  msg.MessageID,
			TopicMsgId: message.MessageID,
		})
		return
	}

	if msg.ReplyToMessage != nil {
		var message model.Msg
		err := DB().Where("topic_id", topic.Id).Where("user_msg_id", msg.ReplyToMessage.MessageID).Find(&message).Error
		if err != nil {
			bot.sendDatabaseError(currentChat, translator, err)
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
		mediaGroupConfig, ok := generateMediaGroup(mediaGroup.Messages, botTopic)
		if !ok {
			bot.sendUnsupportedMessage(currentChat, translator)
			return
		}

		messages, err := bot.SendMediaGroup(mediaGroupConfig)
		if err != nil {
			sendError(err)
			return
		}

		if len(mediaGroup.Messages) != len(messages) {
			bot.sendError(currentChat, translator)
			clog.Errorf("[Bot %d] messages length mismatch, want: %d, got: %d", len(mediaGroup.Messages), len(messages))
			return
		}

		msgs := make([]model.Msg, 0, len(mediaGroup.Messages))
		for i, msg := range mediaGroup.Messages {
			topic_message_id := messages[i].MessageID

			msgs = append(msgs, model.Msg{
				TopicId:    topic.Id,
				UserMsgId:  msg.MessageID,
				TopicMsgId: topic_message_id,
			})
		}

		DB().Create(msgs)
		return
	}

	message, err := bot.Send(botapi.CopyMessageConfig{
		BaseChat:  botTopic,
		FromChat:  currentChatConfig,
		MessageID: msg.MessageID,
	})
	if err != nil {
		sendError(err)
		return
	}

	DB().Create(&model.Msg{
		TopicId:    topic.Id,
		UserMsgId:  msg.MessageID,
		TopicMsgId: message.MessageID,
	})
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

	switch {
	case isServiceMessage(msg):
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: botapi.BaseChatMessage{
				ChatConfig: currentChatConfig,
				MessageID:  msg.MessageID,
			},
		})
		return
	case isIgnoreMessage(msg):
		return
	case isAllowedMessage(msg):
	default:
		bot.sendUnsupportedMessage(currentChat, translator)
		return
	}

	if bot.GroupId == 0 {
		bot.sendNoForwardGroup(currentChat, translator)
		return
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("user_id", msg.From.ID).Find(&topic).Error
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	switch {
	case topic.Id == 0,
		topic.TopicId == 0,
		topic.Verification != model.VerificationCompleted:
		bot.sendFailedToEdit(currentChat, translator)
		return
	case topic.IsBan:
		bot.sendBanned(currentChat, translator)
		return
	}

	var message model.Msg
	err = DB().Where("topic_id", topic.Id).Where("user_msg_id", msg.MessageID).Find(&message).Error
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	if message.Id == 0 {
		bot.sendFailedToEdit(currentChat, translator)
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
		if err, ok := err.(*botapi.Error); ok {
			bot.sendTelegramError(currentChat, err)
			return
		}

		bot.sendError(currentChat, translator)
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

	var isUnsupportMessage bool
	switch {
	case isServiceMessage(msg):
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		return
	case isIgnoreMessage(msg):
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
			bot.sendDatabaseError(currentTopic, translator, err)
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
			bot.sendDatabaseError(currentTopic, translator, err)
			return
		}

		userTranslator := i18n.GetOrDefault(topic.LanguageCode)
		userChat := botapi.BaseChat{
			ChatConfig: botapi.ChatConfig{
				ChatID: topic.UserId,
			},
		}

		bot.sendBanned(userChat, userTranslator)
		bot.sendBanUser(currentTopic, translator, topic.UserId)
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
			bot.sendDatabaseError(currentTopic, translator, err)
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
			bot.sendDatabaseError(currentTopic, translator, err)
			return
		}

		bot.sendUnbanUser(currentTopic, translator, topic.UserId)
		return
	case isAllowedMessage(msg):
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
				bot.sendCommandUsageBan(currentChat, translator)
				return
			}

			bot.bot.RLock()
			defer bot.bot.RUnlock()

			bot.topic.Lock()
			defer bot.topic.Unlock()

			var topic model.Topic
			err = DB().Where("user_id", user_id).Find(&topic).Error
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			if topic.IsBan {
				bot.sendBanUser(currentChat, translator, user_id)
				return
			}

			if topic.TopicId != 0 {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})
				DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)
				topic.TopicId = 0
			}

			topic.UserId = user_id

			err = banTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			bot.sendBanUser(currentChat, translator, user_id)
			return

		case "/unban", "/unban@" + bot.Self.UserName:
			user_id, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				bot.sendCommandUsageUnban(currentChat, translator)
				return
			}

			bot.bot.RLock()
			defer bot.bot.RUnlock()

			bot.topic.Lock()
			defer bot.topic.Unlock()

			var topic model.Topic
			err = DB().Where("user_id", user_id).Find(&topic).Error
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			if topic.Id == 0 {
				bot.sendUnbanUser(currentChat, translator, user_id)
				return
			}

			if topic.TopicId != 0 {
				bot.Request(botapi.ReopenForumTopicConfig{
					BaseForum: currentForum,
				})
			}

			err = unbanTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			bot.sendUnbanUser(currentChat, translator, user_id)
			return

		case "/terminate", "/terminate@" + bot.Self.UserName:
			user_id, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				bot.sendCommandUsageTerminate(currentChat, translator)
				return
			}

			bot.bot.RLock()
			defer bot.bot.RUnlock()

			bot.topic.Lock()
			defer bot.topic.Unlock()

			var topic model.Topic
			err = DB().Where("user_id", user_id).Find(&topic).Error
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			if topic.Id == 0 {
				bot.sendSuccess(currentChat, translator)
				return
			}

			if topic.TopicId != 0 {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})
			}

			err = terminateTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)
			bot.sendSuccess(currentChat, translator)
			return

		default:
			if strings.HasSuffix(command, "@"+bot.Self.UserName) {
				bot.sendUnknownCommand(currentChat, translator)
				return
			}
		}
		return
	}

	mediaGroup, skip := bot.getMediaGroup(msg)
	if skip {
		return
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("topic_id", msg.MessageThreadID).Find(&topic).Error
	if err != nil {
		bot.sendDatabaseError(currentTopic, translator, err)
		return
	}

	if topic.Id == 0 {
		return
	}

	if isUnsupportMessage {
		bot.sendUnsupportedMessage(currentChat, translator)
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
			topic.TopicId = 0

			if topic.IsBan {
				bot.sendBanUser(currentChat, translator, topic.UserId)
				return
			}

			err = banTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			bot.sendBanned(userChat, userTranslator)
			bot.sendBanUser(currentChat, translator, topic.UserId)
			return

		case "/unban", "/unban@" + bot.Self.UserName:
			if !topic.IsBan {
				bot.sendUnbanUser(currentChat, translator, topic.UserId)
				return
			}

			bot.Request(botapi.ReopenForumTopicConfig{
				BaseForum: currentForum,
			})

			err = unbanTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			bot.sendUnbanUser(currentChat, translator, topic.UserId)
			return

		case "/terminate", "/terminate@" + bot.Self.UserName:
			bot.Request(botapi.DeleteForumTopicConfig{
				BaseForum: currentForum,
			})

			err := terminateTopic(&topic)
			if err != nil {
				bot.sendDatabaseError(currentChat, translator, err)
				return
			}

			DB().Model(model.Msg{}).Where("topic_id", topic.Id).Delete(nil)

			if topic.IsBan {
				return
			}

			bot.sendTerminated(userChat, userTranslator)
			return

		default:
			if strings.HasSuffix(command, "@"+bot.Self.UserName) {
				bot.sendUnknownCommand(currentChat, translator)
				return
			}
		}
	}

	if msg.HasProtectedContent {
		bot.sendForwardForbidden(currentChat, translator)
		return
	}

	sendError := func(err error) {
		if err, ok := err.(*botapi.Error); ok {
			if isBlocked(err) {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})
				terminateTopic(&topic)

				bot.sendBlocked(currentGroup, translator, topic.UserId)
				return
			}

			bot.sendTelegramError(currentChat, err)
			return
		}

		bot.sendError(currentChat, translator)
	}

	if topic.Verification != model.VerificationCompleted {
		topic.Verification = model.VerificationCompleted
		saveTopic(&topic)
	}

	if msg.ForwardOrigin != nil {
		if mediaGroup != nil {
			messageIds, err := bot.ForwardMessages(botapi.ForwardMessagesConfig{
				BaseChat:   userChat,
				FromChat:   currentChatConfig,
				MessageIDs: mediaGroup.MessageIds(),
			})
			if err != nil {
				sendError(err)
				return
			}

			var msgs []model.Msg
			for i, msg := range mediaGroup.Messages {
				user_message_id := messageIds[i].MessageID

				msgs = append(msgs, model.Msg{
					TopicId:    topic.Id,
					UserMsgId:  user_message_id,
					TopicMsgId: msg.MessageID,
				})
			}

			DB().Create(msgs)
			return
		}

		message, err := bot.Send(botapi.ForwardConfig{
			BaseChat:  userChat,
			FromChat:  currentChatConfig,
			MessageID: msg.MessageID,
		})
		if err != nil {
			sendError(err)
			return
		}

		DB().Create(&model.Msg{
			TopicId:    topic.Id,
			UserMsgId:  message.MessageID,
			TopicMsgId: msg.MessageID,
		})
		return
	}

	if msg.ReplyToMessage != nil && msg.ReplyToMessage.MessageID != msg.MessageThreadID {
		var message model.Msg
		err := DB().Where("topic_id", topic.Id).Where("topic_msg_id", msg.ReplyToMessage.MessageID).Find(&message).Error
		if err != nil {
			bot.sendDatabaseError(currentTopic, translator, err)
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
		mediaGroupConfig, ok := generateMediaGroup(mediaGroup.Messages, userChat)
		if !ok {
			bot.sendUnsupportedMessage(currentChat, translator)
			return
		}

		messages, err := bot.SendMediaGroup(mediaGroupConfig)
		if err != nil {
			sendError(err)
			return
		}

		var msgs []model.Msg
		for i, msg := range mediaGroup.Messages {
			user_message_id := messages[i].MessageID

			msgs = append(msgs, model.Msg{
				TopicId:    topic.Id,
				UserMsgId:  user_message_id,
				TopicMsgId: msg.MessageID,
			})
		}

		DB().Create(msgs)
		return
	}

	message, err := bot.Send(botapi.CopyMessageConfig{
		BaseChat:  userChat,
		FromChat:  currentChatConfig,
		MessageID: msg.MessageID,
	})
	if err != nil {
		sendError(err)
		return
	}

	DB().Create(&model.Msg{
		TopicId:    topic.Id,
		UserMsgId:  message.MessageID,
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
	currentMessage := botapi.BaseChatMessage{
		ChatConfig: currentChatConfig,
		MessageID:  msg.MessageID,
	}

	var isUnsupportMessage bool
	switch {
	case isServiceMessage(msg):
		bot.Request(botapi.DeleteMessageConfig{
			BaseChatMessage: currentMessage,
		})
		return

	case isIgnoreMessage(msg):
		return

	case isAllowedMessage(msg):
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

	currentChat := botapi.BaseChat{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
		ReplyParameters: botapi.ReplyParameters{
			MessageID: msg.MessageID,
		},
	}

	bot.bot.RLock()
	defer bot.bot.RUnlock()

	bot.topic.Lock()
	defer bot.topic.Unlock()

	var topic model.Topic
	err := DB().Where("topic_id", msg.MessageThreadID).Find(&topic).Error
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	if topic.Id == 0 {
		return
	}

	if isUnsupportMessage {
		bot.sendUnsupportedMessage(currentChat, translator)
		return
	}

	var message model.Msg
	err = DB().Where("topic_id", topic.Id).Where("topic_msg_id", msg.MessageID).Find(&message).Error
	if err != nil {
		bot.sendDatabaseError(currentChat, translator, err)
		return
	}

	if message.Id == 0 {
		bot.sendFailedToEdit(currentChat, translator)
		return
	}

	currentForum := botapi.BaseForum{
		ChatConfig:      currentChatConfig,
		MessageThreadID: msg.MessageThreadID,
	}
	currentGroup := botapi.BaseChat{
		ChatConfig: currentChatConfig,
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
		if err, ok := err.(*botapi.Error); ok {
			if isBlocked(err) {
				bot.Request(botapi.DeleteForumTopicConfig{
					BaseForum: currentForum,
				})
				terminateTopic(&topic)

				bot.sendBlocked(currentGroup, translator, topic.UserId)
				return
			}

			bot.sendTelegramError(currentChat, err)
			return
		}

		bot.sendError(currentChat, translator)
		return
	}
}
