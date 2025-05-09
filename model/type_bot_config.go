package model

type BotConfig struct {
	Token        string
	GroupId      int64
	LanguageCode string

	WebHook struct {
		Host string
	}
}
