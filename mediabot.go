package mediabot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"github.com/looplab/fsm"
	"github.com/xdefrag/mediabot/db"
)

type Mediabot struct {
	l   *slog.Logger
	bot BotSender
	q   Querier
}

func New(
	l *slog.Logger,
	bot BotSender,
	q Querier,
) *Mediabot {
	return &Mediabot{
		l:   l,
		bot: bot,
		q:   q,
	}
}

const FeedbackChatID int64 = -4129350106

func (m *Mediabot) Handle(ctx context.Context, upd tgbotapi.Update) error {
	sm := fsm.NewFSM(
		StateInit,
		fsm.Events{
			{Name: EventStart, Src: []string{StateInit}, Dst: StateMain},
			{Name: EventMainMessage, Src: []string{StateMain}, Dst: StateMain},
		},
		fsm.Callbacks{
			EventStart: func(ctx context.Context, e *fsm.Event) {
				state := e.Args[0].(db.State)

				if err := m.q.CreateState(ctx, db.CreateStateParams{
					UserID: state.UserID,
					State:  StateMain,
					Data:   state.Data,
					Meta:   state.Meta,
				}); err != nil {
					e.Cancel(err)
					return
				}

				_, err := m.bot.Send(tgbotapi.NewMessage(state.UserID, "works"))
				if err != nil {
					e.Cancel(err)
				}
			},
			EventMainMessage: func(ctx context.Context, e *fsm.Event) {
				state := e.Args[0].(db.State)

				if err := m.q.CreateState(ctx, db.CreateStateParams{
					UserID: state.UserID,
					State:  StateMain,
					Data:   state.Data,
					Meta:   state.Meta,
				}); err != nil {
					e.Cancel(err)
					return
				}

				// check message with emoji

				if _, err := m.bot.Send(
					tgbotapi.NewForward(FeedbackChatID, state.UserID, state.Data["message_id"].(int)),
				); err != nil {
					e.Cancel(err)
				}
			},
		},
	)

	if upd.Message == nil {
		return nil
	}

	if upd.Message.Chat.IsGroup() {
		if upd.Message.ReplyToMessage == nil || upd.Message.ReplyToMessage.ForwardFrom == nil {
			return nil
		}

		if err := m.q.CreateResponse(ctx, db.CreateResponseParams{
			FromUserID: upd.Message.Chat.ID,
			ToUserID:   upd.Message.ReplyToMessage.ForwardFrom.ID,
			Message:    upd.Message.Text,
		}); err != nil {
			return err
		}

		if _, err := m.bot.Send(
			tgbotapi.NewForward(upd.Message.ReplyToMessage.ForwardFrom.ID, upd.Message.Chat.ID, upd.Message.MessageID),
		); err != nil {
			return err
		}
	}

	if upd.Message.Chat.IsPrivate() {
		if upd.Message.From == nil || upd.Message.Chat == nil {
			return nil
		}

		state, err := m.q.GetState(ctx, upd.Message.From.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			state = db.State{
				UserID: upd.Message.From.ID,
				State:  StateInit,
				Data:   make(map[string]interface{}),
				Meta:   make(map[string]interface{}),
			}
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		sm.SetState(state.State)

		state.Data["message"] = upd.Message.Text
		state.Data["message_id"] = upd.Message.MessageID
		state.Meta["username"] = upd.Message.From.UserName
		state.Meta["firstname"] = upd.Message.From.FirstName
		state.Meta["lastname"] = upd.Message.From.LastName
		state.Meta["chat_type"] = upd.Message.Chat.Type
		state.Meta["chat_title"] = upd.Message.Chat.Title
		state.Meta["chat_id"] = upd.Message.Chat.ID

		event := ""

		if strings.HasPrefix(upd.Message.Text, "/") {
			event = upd.Message.Text[1:]
		}

		if !strings.HasPrefix(upd.Message.Text, "/") {
			event = fmt.Sprintf("%s_message", state.State)
		}

		if err := sm.Event(ctx, event, state); err != nil && !errors.Is(err, fsm.NoTransitionError{}) {
			return err
		}
	}

	return nil
}
