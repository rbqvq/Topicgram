package bots

import (
	"Topicgram/i18n"
	"Topicgram/model"
	"Topicgram/utils"
	"fmt"
	"net/http"
	"net/url"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/gin-gonic/gin"
	"gitlab.com/CoiaPrant/clog"
)

var (
	bot         *Bot
	secretToken string
)

func Load(botConfig *model.BotConfig) error {
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
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusBadRequest, gin.H{"error": "bot not found"})
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
