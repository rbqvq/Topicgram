package bots

import (
	"Topicgram/i18n"
	"Topicgram/model"
	"Topicgram/utils"
	"context"
	"fmt"
	"time"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/clog"
)

var (
	bot      *Bot
	shutdown context.CancelFunc
)

func Load(botConfig *model.BotConfig) error {
	options := make([]botapi.BotAPIOption, 0, 3)
	options = append(options, botapi.WithHTTPClient(utils.BotClient))
	if clog.Level() == clog.LevelDebug {
		options = append(options,
			botapi.WithDebug(true),
			botapi.WithLogger(clog.Printer(2, "Bot", clog.LevelDebug)),
		)
	} else {
		options = append(options, botapi.WithLoggingDisabled())
	}

	b, err := botapi.NewBotAPIWithOptions(botConfig.Token, options...)
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

	ctx, cancel := context.WithCancel(context.Background())
	shutdown = cancel
	go getUpdates(ctx, botapi.UpdateConfig{})

	clog.Success("[Bot] Load completed")
	return nil
}

func Shutdown() {
	shutdown()
	clog.Infof("[Bot] Shutdown")
}

func getUpdates(ctx context.Context, config botapi.UpdateConfig) {
	for {
		updates, err := bot.GetUpdatesWithContext(ctx, config)
		if err != nil {
			if ctx.Err() == nil {
				clog.Errorf("Failed to get updates (%s), retrying in 3 seconds...", err)
				time.Sleep(time.Second * 3)
				continue
			}
			return
		}

		for _, update := range updates {
			if update.UpdateID >= config.Offset {
				config.Offset = update.UpdateID + 1
				go bot.handleUpdate(&update)
			}
		}
	}
}
