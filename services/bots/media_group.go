package bots

import (
	"sort"
	"sync"
	"time"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/CoiaPrant/cache2go"
)

const mediaGroupLifeSpan = 3 * time.Second

type (
	mediaGroupCache = *cache2go.CacheTableOf[string, *MediaGroup]
	mediaGroupItem  = *cache2go.CacheItemOf[string, *MediaGroup]
)

func NewMediaGroupCache() mediaGroupCache {
	return cache2go.CacheOf[string, *MediaGroup]()
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

func (bot *Bot) getMediaGroup(msg *botapi.Message) (*MediaGroup, bool) {
	if msg.MediaGroupID == "" {
		return nil, false
	}

	mediaGroup := &MediaGroup{}
	if bot.mediaGroups.NotFoundAdd(msg.MediaGroupID, mediaGroupLifeSpan, mediaGroup) {
		done := make(chan struct{})
		mediaGroup.done = done
		mediaGroup.Add(msg)

		<-done
		mediaGroup.Sort()
		return mediaGroup, false
	}

	item, err := bot.mediaGroups.Value(msg.MediaGroupID)
	if err == nil {
		mediaGroup := item.Data()
		mediaGroup.Add(msg)
		return nil, true
	}

	return nil, false
}

func generateMediaGroup(msgs []*botapi.Message, baseChat botapi.BaseChat) (botapi.MediaGroupConfig, bool) {
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
			return botapi.MediaGroupConfig{}, false
		}

		medias = append(medias, inputMedia)
	}

	return botapi.MediaGroupConfig{
		BaseChat: baseChat,
		Media:    medias,
	}, true
}
