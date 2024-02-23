package mediabot_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xdefrag/mediabot"
	"github.com/xdefrag/mediabot/db"
	"github.com/xdefrag/mediabot/mocks"
)

func TestMediabot_Private(t *testing.T) {
	const (
		userID    int64 = 123
		messageID int   = 321
	)

	tests := []struct {
		text             string
		state            string
		expectedNewState string
		sentMatchedBy    interface{}
	}{
		{
			text:             "/start",
			state:            mediabot.StateInit,
			expectedNewState: mediabot.StateMain,
			sentMatchedBy: func(msg tgbotapi.MessageConfig) bool {
				return assert.Equal(t, "works", msg.Text)
			},
		},
		{
			text:             "Hey i want to see some media",
			state:            mediabot.StateMain,
			expectedNewState: mediabot.StateMain,
			sentMatchedBy: func(msg tgbotapi.ForwardConfig) bool {
				return assert.Equal(t, userID, msg.FromChatID) &&
					assert.Equal(t, mediabot.FeedbackChatID, msg.ChatID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.state, tt.text), func(t *testing.T) {
			ctx := context.Background()
			l := slog.New(slog.NewJSONHandler(os.Stderr, nil))

			bot := mocks.NewBotSender(t)
			q := mocks.NewQuerier(t)

			upd := tgbotapi.Update{
				Message: &tgbotapi.Message{
					MessageID: messageID,
					From: &tgbotapi.User{
						ID:        userID,
						UserName:  "username",
						FirstName: "firstname",
						LastName:  "lastname",
					},
					Text: tt.text,
					Chat: &tgbotapi.Chat{
						ID:    userID,
						Title: "chat_title",
						Type:  "private",
					},
				},
			}

			state := db.State{
				UserID: userID,
				State:  tt.state,
				Data:   make(map[string]interface{}),
				Meta:   make(map[string]interface{}),
			}

			q.EXPECT().GetState(ctx, userID).Return(state, lo.Ternary(tt.state != "", nil, pgx.ErrNoRows))

			q.EXPECT().CreateState(mock.AnythingOfType("*context.cancelCtx"), db.CreateStateParams{
				UserID: userID,
				State:  tt.expectedNewState,
				Data: map[string]interface{}{
					"message":    tt.text,
					"message_id": messageID,
				},
				Meta: map[string]interface{}{
					"chat_type":  "private",
					"chat_title": "chat_title",
					"chat_id":    userID,
					"username":   "username",
					"firstname":  "firstname",
					"lastname":   "lastname",
				},
			}).Return(nil)

			bot.EXPECT().Send(mock.MatchedBy(tt.sentMatchedBy)).Return(tgbotapi.Message{}, nil)

			err := mediabot.New(l, bot, q).Handle(ctx, upd)
			require.NoError(t, err)
		})
	}
}

func TestMediabot_Group(t *testing.T) {
	const (
		fromChatID int64 = 123
		toUserID   int64 = 456
		messageID  int   = 321
		text             = "test"
	)

	ctx := context.Background()
	l := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	bot := mocks.NewBotSender(t)
	q := mocks.NewQuerier(t)

	upd := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: messageID,
			Text:      text,
			Chat: &tgbotapi.Chat{
				ID:   fromChatID,
				Type: "group",
			},
			ReplyToMessage: &tgbotapi.Message{
				ForwardFrom: &tgbotapi.User{
					ID: toUserID,
				},
			},
		},
	}

	q.EXPECT().CreateResponse(ctx, db.CreateResponseParams{
		FromUserID: fromChatID,
		ToUserID:   toUserID,
		Message:    text,
	}).Return(nil)

	bot.EXPECT().Send(mock.MatchedBy(func(msg tgbotapi.ForwardConfig) bool {
		return assert.Equal(t, fromChatID, msg.FromChatID) &&
			assert.Equal(t, toUserID, msg.ChatID) &&
			assert.Equal(t, messageID, msg.MessageID)
	})).Return(tgbotapi.Message{}, nil)

	err := mediabot.New(l, bot, q).Handle(ctx, upd)
	require.NoError(t, err)
}
