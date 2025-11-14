package bots

import (
	botapi "github.com/OvyFlash/telegram-bot-api"
)

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

			baseInputMedia.Type = "document"
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
