package bot

import (
	"sync"

	botapi "github.com/OvyFlash/telegram-bot-api"
)

type BotAPI struct {
	*botapi.BotAPI
	bot   sync.RWMutex
	topic sync.Mutex
}
