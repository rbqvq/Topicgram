package bot

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/cache2go"
)

type BotAPI struct {
	*botapi.BotAPI
	bot         sync.RWMutex
	topic       sync.Mutex
	mediaGroups mediaGroupCache
}

// ForwardMessages forwards multi-messages and returns the resulting message ids.
func (bot *BotAPI) ForwardMessages(c botapi.ForwardMessagesConfig) ([]botapi.MessageID, error) {
	resp, err := bot.Request(c)
	if err != nil {
		return nil, err
	}

	var message_ids []botapi.MessageID
	err = json.Unmarshal(resp.Result, &message_ids)
	return message_ids, err
}

type MediaGroup struct {
	sync.Mutex
	Messages []*botapi.Message
	done     chan struct{}
}

func (mediaGroup *MediaGroup) Add(msg *botapi.Message) {
	mediaGroup.Lock()
	defer mediaGroup.Unlock()

	mediaGroup.Messages = append(mediaGroup.Messages, msg)
}

func (mediaGroup *MediaGroup) MessageIds() []int {
	mediaGroup.Lock()
	defer mediaGroup.Unlock()

	message_ids := make([]int, 0, len(mediaGroup.Messages))
	for _, msg := range mediaGroup.Messages {
		message_ids = append(message_ids, msg.MessageID)
	}

	return message_ids
}

func (mediaGroup *MediaGroup) Sort() {
	mediaGroup.Lock()
	defer mediaGroup.Unlock()

	if len(mediaGroup.Messages) <= 1 {
		return
	}

	sort.Slice(mediaGroup.Messages, func(i, j int) bool {
		return mediaGroup.Messages[i].MessageID < mediaGroup.Messages[j].MessageID
	})
}

const mediaGroupLifeSpan = 3 * time.Second

type (
	mediaGroupCache = *cache2go.CacheTableOf[string, *MediaGroup]
	mediaGroupItem  = *cache2go.CacheItemOf[string, *MediaGroup]
)

func NewMediaGroupCache() mediaGroupCache {
	return cache2go.CacheOf[string, *MediaGroup]()
}
