package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"github.com/peterbourgon/ff/v4"
	"github.com/xdefrag/mediabot"
	"github.com/xdefrag/mediabot/db"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	l := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	mediabotfs := ff.NewFlagSet("mediabot")
	servefs := ff.NewFlagSet("serve").SetParent(mediabotfs)
	telegramToken := servefs.String('t', "telegram-token", "", "Telegram bot token")
	postgresDSN := servefs.String('d', "postgres", "postgresql://localhost:5432/postgres", "Postgres dsn")

	ff.Parse(servefs, os.Args[1:], ff.WithEnvVarPrefix("MEDIABOT"))

	bot, err := tgbotapi.NewBotAPI(*telegramToken)
	if err != nil {
		l.ErrorContext(ctx, "failed to create bot", "error", err)
		os.Exit(1)
	}

	conn, err := pgx.Connect(ctx, *postgresDSN)
	if err != nil {
		l.ErrorContext(ctx, "failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	m := mediabot.New(l, bot, db.New(conn))
	updchan := bot.GetUpdatesChan(tgbotapi.NewUpdate(0))

	for {
		select {
		case upd := <-updchan:
			if err := m.Handle(ctx, upd); err != nil {
				l.ErrorContext(ctx, "failed to handle update", "error", err)
			}
		case <-ctx.Done():
			os.Exit(0)
		}
	}
}
