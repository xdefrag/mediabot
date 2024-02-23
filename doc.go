package mediabot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xdefrag/mediabot/db"
)

const (
	StateInit = "init"
	StateMain = "main"
)

const (
	EventStart       = "start"
	EventMainMessage = "main_message"
)

//go:generate mockery --case=underscore --with-expecter --name=BotSender
type BotSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

//go:generate mockery --case=underscore --with-expecter --name=Querier
type Querier interface {
	db.Querier
}
