package iambot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/gopherex/xlog"
)

// Callback data prefixes for the inline decision buttons.
const (
	callbackApprove = "ar:app:"
	callbackDeny    = "ar:deny:"

	// pendingListLimit bounds one /pending listing.
	pendingListLimit = 10
)

const helpText = `IAM access-request bot.

/pending — list pending access requests with decision buttons
/invite <email> — mint an email-bound invitation (token shown once)

New pending requests are pushed here automatically. Approving sends the
requester an email with a magic sign-up link; the invite token from /invite
is emailed too and shown here exactly once.`

// Bot wires the Telegram bot, the IAM client and the seen-state together.
type Bot struct {
	cfg      Config
	iam      *IAM
	state    *State
	log      *xlog.Logger
	telegram *tgbot.Bot
}

// New builds the Telegram bot with its handlers registered.
func New(cfg Config, iam *IAM, state *State, log *xlog.Logger) (*Bot, error) {
	b := &Bot{cfg: cfg, iam: iam, state: state, log: log}

	// WithSkipGetMe: construction must not touch the network, so an IAM-bot
	// restart survives a transient Telegram outage (long polling reconnects
	// on its own) instead of crash-looping on getMe.
	instance, err := tgbot.New(cfg.TelegramToken,
		tgbot.WithSkipGetMe(),
		tgbot.WithDefaultHandler(b.defaultHandler))
	if err != nil {
		return nil, fmt.Errorf("telegram bot: %w", err)
	}

	instance.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "ar:", tgbot.MatchTypePrefix, b.onCallback)

	b.telegram = instance

	return b, nil
}

// Start runs the Telegram long polling until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) {
	b.telegram.Start(ctx)
}

// userID extracts the acting telegram user from an update (0 when unknown).
func userID(upd *models.Update) int64 {
	switch {
	case upd.Message != nil:
		return upd.Message.From.ID
	case upd.CallbackQuery != nil:
		return upd.CallbackQuery.From.ID
	default:
		return 0
	}
}

// defaultHandler is the allowlist gate + command router.
func (b *Bot) defaultHandler(ctx context.Context, _ *tgbot.Bot, upd *models.Update) {
	if upd.Message == nil {
		return
	}

	if !b.cfg.Allows(upd.Message.From.ID) {
		return // silent: strangers get no signal that the bot is alive
	}

	text := strings.TrimSpace(upd.Message.Text)
	cmd, rest, _ := strings.Cut(text, " ")
	cmd, _, _ = strings.Cut(cmd, "@") // tolerate /pending@botname in groups

	switch cmd {
	case "/start", "/help":
		b.reply(ctx, upd.Message.Chat.ID, helpText)
	case "/pending":
		b.cmdPending(ctx, upd.Message.Chat.ID)
	case "/invite":
		b.cmdInvite(ctx, upd.Message.Chat.ID, strings.TrimSpace(rest))
	default:
		b.reply(ctx, upd.Message.Chat.ID, "Unknown command:\n\n"+helpText)
	}
}

func (b *Bot) reply(ctx context.Context, chatID int64, text string) {
	_, _ = b.telegram.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: text})
}

// cmdPending lists the current backlog with decision buttons.
func (b *Bot) cmdPending(ctx context.Context, chatID int64) {
	pending, err := b.iam.ListPending(ctx)
	if err != nil {
		b.reply(ctx, chatID, "Failed to list access requests: "+err.Error())

		return
	}

	if len(pending) == 0 {
		b.reply(ctx, chatID, "No pending access requests.")

		return
	}

	shown := pending
	if len(shown) > pendingListLimit {
		shown = shown[:pendingListLimit]
	}

	for i := range shown {
		if _, err := b.telegram.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      chatID,
			Text:        requestCard(&shown[i]),
			ReplyMarkup: decisionKeyboard(shown[i].ID),
		}); err != nil {
			b.reply(ctx, chatID, "Failed to render request: "+err.Error())

			return
		}
	}

	if rest := len(pending) - len(shown); rest > 0 {
		b.reply(ctx, chatID, fmt.Sprintf("…and %d more (press the buttons above; re-run /pending after deciding).", rest))
	}
}

// cmdInvite mints an email-bound invitation and shows the raw token once.
func (b *Bot) cmdInvite(ctx context.Context, chatID int64, email string) {
	if !strings.Contains(email, "@") || strings.Contains(email, " ") {
		b.reply(ctx, chatID, "Usage: /invite <email>")

		return
	}

	token, err := b.iam.CreateInvite(ctx, email)
	if err != nil {
		b.reply(ctx, chatID, "Failed to create invite: "+err.Error())

		return
	}

	// The requester also gets the invitation email from IAM; the token is
	// shown in the chat exactly once, like the admin panel does.
	b.reply(ctx, chatID,
		"Invitation for "+email+" created.\nToken (shown once):\n"+token)
}

// onCallback handles the inline ✅/❌ buttons.
func (b *Bot) onCallback(ctx context.Context, _ *tgbot.Bot, upd *models.Update) {
	if upd.CallbackQuery == nil {
		return
	}

	query := upd.CallbackQuery

	if !b.cfg.Allows(query.From.ID) {
		_, _ = b.telegram.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Not authorized.",
		})

		return
	}

	var (
		id     string
		action string
		err    error
	)

	switch {
	case strings.HasPrefix(query.Data, callbackApprove):
		id = strings.TrimPrefix(query.Data, callbackApprove)
		action = "approved"
		err = b.iam.Approve(ctx, id)
	case strings.HasPrefix(query.Data, callbackDeny):
		id = strings.TrimPrefix(query.Data, callbackDeny)
		action = "denied"
		err = b.iam.Deny(ctx, id, "")
	default:
		return
	}

	switch {
	case err == nil:
		// fall through to edit
	case errors.Is(err, ErrNotFound):
		b.answerCallback(ctx, query.ID, "Already decided elsewhere.")
		b.editCard(ctx, query, "⏭ request "+shortID(id)+" was already decided elsewhere")
		b.log.Info("decision: already decided elsewhere", idField(id), operatorField(query.From.ID))

		return
	default:
		b.answerCallback(ctx, query.ID, "Failed: "+err.Error())
		b.log.Error("decision failed", idField(id), operatorField(query.From.ID), errField(err))

		return
	}

	mark := "✅"
	if action == "denied" {
		mark = "❌"
	}

	b.answerCallback(ctx, query.ID, mark+" "+action)
	b.editCard(ctx, query, mark+" request "+shortID(id)+" "+action+" — decision email sent")
	b.log.Info("decision applied", idField(id), xlog.Int64("operator", query.From.ID), xlog.String("action", action))
}

func (b *Bot) answerCallback(ctx context.Context, cqID, text string) {
	_, _ = b.telegram.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cqID,
		Text:            text,
	})
}

// editCard rewrites the card message the button belonged to.
func (b *Bot) editCard(ctx context.Context, query *models.CallbackQuery, text string) {
	msg := query.Message.Message
	if msg == nil {
		return
	}

	_, _ = b.telegram.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Text:      text,
	})
}

// Announce pushes a new request card to every allowed operator.
func (b *Bot) Announce(ctx context.Context, req *AccessRequest) error {
	var firstErr error
	for _, uid := range b.cfg.AllowedUsers {
		if _, err := b.telegram.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      uid,
			Text:        "📬 New access request\n\n" + requestCard(req),
			ReplyMarkup: decisionKeyboard(req.ID),
		}); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if firstErr != nil {
		b.log.Warn("announce failed", idField(req.ID), errField(firstErr))
	} else {
		b.log.Info("announced", idField(req.ID), xlog.String("email", req.Email))
	}

	return firstErr
}

// requestCard renders one request as plain text (no markdown escaping worries).
func requestCard(r *AccessRequest) string {
	var buf strings.Builder
	buf.WriteString("Email: ")
	buf.WriteString(r.Email)

	if r.Reason != "" {
		buf.WriteString("\nReason: ")
		buf.WriteString(r.Reason)
	}

	buf.WriteString("\nID: ")
	buf.WriteString(r.ID)

	return buf.String()
}

// decisionKeyboard builds the inline ✅/❌ pair for a request id.
func decisionKeyboard(id string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: "✅ Approve", CallbackData: callbackApprove + id},
			{Text: "❌ Deny", CallbackData: callbackDeny + id},
		}},
	}
}

// shortIDLen is how much of a request id confirmation messages show.
const shortIDLen = 8

// Log field helpers shared across the package.
func idField(id string) xlog.Field      { return xlog.String("request_id", id) }
func errField(err error) xlog.Field     { return xlog.Error("err", err) }
func operatorField(id int64) xlog.Field { return xlog.Int64("operator", id) }

// shortID renders a short id prefix for confirmation messages.
func shortID(id string) string {
	if len(id) > shortIDLen {
		return id[:shortIDLen]
	}

	return id
}
