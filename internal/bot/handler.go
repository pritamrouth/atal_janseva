// Package bot implements the Ataljanseva onboarding conversation flow.
// All operations are context-aware for proper timeout/cancellation propagation.
package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ataljanseva/whatsapp-bot/internal/db"
	"github.com/ataljanseva/whatsapp-bot/internal/inactivity"
	"github.com/ataljanseva/whatsapp-bot/internal/store"
	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
)

// jobTimeout is the max time a single message may take end-to-end.
const jobTimeout = 12 * time.Second

// Handler processes inbound WhatsApp messages.
type Handler struct {
	wa       *whatsapp.Client
	store    *store.Store
	repo     *db.Repo
	inactive *inactivity.Monitor
}

// New returns a Handler wired with the inactivity monitor.
func New(wa *whatsapp.Client, s *store.Store, repo *db.Repo, inactive *inactivity.Monitor) *Handler {
	return &Handler{wa: wa, store: s, repo: repo, inactive: inactive}
}

// HandleRaw is the worker entry-point: one call per inbound message.
func (h *Handler) HandleRaw(msg whatsapp.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	log := slog.With("phone", msg.From, "type", msg.Type)
	phone := msg.From

	// ── Reset inactivity clock on every inbound message ───────────────────────
	h.inactive.Touch(ctx, phone)

	sess, err := h.store.Get(ctx, phone)
	if err != nil {
		log.Error("get session", "err", err)
		return
	}

	// ── Global reset command ──────────────────────────────────────────────────
	if msg.Type == "text" && strings.EqualFold(strings.TrimSpace(msg.Text.Body), "reset") {
		if err := h.store.Reset(ctx, phone); err != nil {
			log.Error("reset session", "err", err)
		}
		h.inactive.Cancel(ctx, phone)
		_ = h.wa.SendText(ctx, phone, "🔄 Session reset. Type anything to start over.")
		return
	}

	switch sess.Step {

	case store.StepStart:
		if msg.Type == "interactive" && msg.Interactive != nil {
			id := buttonID(msg.Interactive)
			switch id {
			case "lang_en":
				sess.Lang = "en"
			case "lang_mr":
				sess.Lang = "mr"
			case "lang_hi":
				sess.Lang = "hi"
			default:
				h.sendLanguagePicker(ctx, phone, phone)
				return
			}
			sess.Step = store.StepLangChosen
			if err := h.store.Save(ctx, sess); err != nil {
				log.Error("save session", "err", err)
				return
			}
			_ = h.wa.SendText(ctx, phone, h.t(sess).PinPrompt)
		} else {
			h.sendLanguagePicker(ctx, phone, phone) // TASK 1: pass real phone
		}

	case store.StepLangChosen:
		if msg.Type == "text" {
			h.handlePin(ctx, phone, sess, strings.TrimSpace(msg.Text.Body))
		} else {
			_ = h.wa.SendText(ctx, phone, h.t(sess).PinPrompt)
		}

	case store.StepWardChosen:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleWardReply(ctx, phone, sess, msg.Interactive)
		} else {
			h.promptWard(ctx, phone, sess)
		}

	case store.StepNagarsevak:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleNagarsevakReply(ctx, phone, sess, msg.Interactive)
		} else {
			h.promptNagarsevak(ctx, phone, sess)
		}

	case store.StepMainMenu:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleMainMenuSelection(ctx, phone, sess, msg.Interactive)
		} else {
			h.sendMainMenu(ctx, phone, sess)
		}
	}
}

// ─────────────────────────────────────────────
// Step 0 – language picker
// ─────────────────────────────────────────────

// sendLanguagePicker sends the greeting with the user's real phone number (TASK 1).
func (h *Handler) sendLanguagePicker(ctx context.Context, phone, rawPhone string) {
	// Use "en" greeting so the initial message is always in English.
	// The phone number is injected dynamically via GreetingFor().
	greeting := GreetingFor("en", rawPhone)

	err := h.wa.SendButtons(ctx, phone, greeting, [][2]string{
		{"lang_en", "🇮🇳 English"},
		{"lang_mr", "🇮🇳 मराठी"},
		{"lang_hi", "🇮🇳 हिंदी"},
	})
	if err != nil {
		slog.Error("sendLanguagePicker", "phone", phone, "err", err)
	}
}

// ─────────────────────────────────────────────
// Step 1 – PIN → DB
// ─────────────────────────────────────────────

func (h *Handler) handlePin(ctx context.Context, phone string, sess *store.Session, pin string) {
	if len(pin) != 6 {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}
	loc, err := h.repo.LocationByPincode(ctx, pin)
	if err == sql.ErrNoRows {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}
	if err != nil {
		slog.Error("LocationByPincode", "pin", pin, "err", err)
		_ = h.wa.SendText(ctx, phone, "⚠️ Database error, please try again shortly.")
		return
	}
	sess.Pincode = pin
	sess.State = loc.State
	sess.District = loc.District
	sess.Step = store.StepWardChosen
	if err := h.store.Save(ctx, sess); err != nil {
		slog.Error("save session", "phone", phone, "err", err)
		return
	}
	h.promptWard(ctx, phone, sess)
}

// ─────────────────────────────────────────────
// Step 2 – ward list
// ─────────────────────────────────────────────

func (h *Handler) promptWard(ctx context.Context, phone string, sess *store.Session) {
	wards, err := h.repo.WardsByPincode(ctx, sess.Pincode)
	if err != nil {
		slog.Error("WardsByPincode", "pin", sess.Pincode, "err", err)
		_ = h.wa.SendText(ctx, phone, "⚠️ Could not load ward data. Please try again.")
		return
	}
	if len(wards) == 0 {
		_ = h.wa.SendText(ctx, phone, "⚠️ No wards found for this PIN. Please re-enter your PIN.")
		sess.Step = store.StepLangChosen
		_ = h.store.Save(ctx, sess)
		return
	}

	t := h.t(sess)
	rows := make([][3]string, 0, len(wards))
	for _, w := range wards {
		label := w.Code
		if sess.Lang != "en" && w.CodeHindi != "" {
			label = w.Code + " (" + w.CodeHindi + ")"
		}
		rows = append(rows, [3]string{"ward_" + w.Code, label, ""})
	}

	bodyText := fmt.Sprintf(t.WardPrompt, sess.State, sess.District)
	if err := h.wa.SendList(ctx, phone, bodyText, "📍 Select Ward", []whatsapp.ListSection{
		{Title: "🏙 Available Wards", Rows: rows},
	}); err != nil {
		slog.Error("promptWard SendList", "phone", phone, "err", err)
	}
}

func (h *Handler) handleWardReply(ctx context.Context, phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	id := listID(ir)
	if !strings.HasPrefix(id, "ward_") {
		h.promptWard(ctx, phone, sess)
		return
	}
	sess.Ward = strings.TrimPrefix(id, "ward_")
	sess.Step = store.StepNagarsevak
	if err := h.store.Save(ctx, sess); err != nil {
		slog.Error("save session", "phone", phone, "err", err)
		return
	}
	h.promptNagarsevak(ctx, phone, sess)
}

// ─────────────────────────────────────────────
// Step 3 – nagarsevak list (TASK 2: profile photo)
// ─────────────────────────────────────────────

func (h *Handler) promptNagarsevak(ctx context.Context, phone string, sess *store.Session) {
	nagarsevaks, err := h.repo.NagarsevaksByWard(ctx, sess.Pincode, sess.Ward)
	if err != nil {
		slog.Error("NagarsevaksByWard", "pin", sess.Pincode, "ward", sess.Ward, "err", err)
		_ = h.wa.SendText(ctx, phone, "⚠️ Could not load nagarsevak data. Please try again.")
		return
	}
	if len(nagarsevaks) == 0 {
		_ = h.wa.SendText(ctx, phone, "⚠️ No nagarsevaks found for Ward "+sess.Ward+". Please choose a different ward.")
		sess.Step = store.StepWardChosen
		_ = h.store.Save(ctx, sess)
		h.promptWard(ctx, phone, sess)
		return
	}

	t := h.t(sess)

	// TASK 2 – Send profile photo for each nagarsevak before the list.
	// WhatsApp does not support images inside list rows, so we send each
	// candidate's photo as an image message immediately before the selection
	// list. This gives the user a visual reference while they choose.
	for _, ns := range nagarsevaks {
		if ns.ProfilePhoto != "" {
			name := ns.FullName
			if sess.Lang != "en" && ns.NameHindi != "" {
				name = ns.NameHindi
			}
			caption := fmt.Sprintf("🏅 *%s*\n🎖 %s · 🏙 Ward %s", name, ns.Party, ns.Ward)
			if sendErr := h.wa.SendImage(ctx, phone, ns.ProfilePhoto, caption); sendErr != nil {
				// Non-fatal: log and continue — the list will still be sent
				slog.Warn("promptNagarsevak: failed to send profile photo",
					"phone", phone, "nagarsevak_id", ns.ID, "err", sendErr)
			}
		}
	}

	// Build the selection list
	rows := make([][3]string, 0, len(nagarsevaks))
	for _, ns := range nagarsevaks {
		name := ns.FullName
		if sess.Lang != "en" && ns.NameHindi != "" {
			name = ns.NameHindi
		}
		rows = append(rows, [3]string{
			"ns_" + ns.ID,
			name,
			ns.Party + " · Ward " + ns.Ward,
		})
	}

	if err := h.wa.SendList(ctx, phone, t.NagarsevakPrompt, "🏅 Select Nagarsevak", []whatsapp.ListSection{
		{Title: "👥 Candidates", Rows: rows},
	}); err != nil {
		slog.Error("promptNagarsevak SendList", "phone", phone, "err", err)
	}
}

func (h *Handler) handleNagarsevakReply(ctx context.Context, phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	id := listID(ir)
	if !strings.HasPrefix(id, "ns_") {
		h.promptNagarsevak(ctx, phone, sess)
		return
	}
	nsID := strings.TrimPrefix(id, "ns_")
	ns, err := h.repo.NagarsevakByID(ctx, nsID)
	if err != nil {
		slog.Error("NagarsevakByID", "id", nsID, "err", err)
		_ = h.wa.SendText(ctx, phone, "⚠️ Could not find the selected nagarsevak. Please try again.")
		h.promptNagarsevak(ctx, phone, sess)
		return
	}
	sess.NagarsevakID = ns.ID
	sess.NagarsevakName = ns.FullName
	sess.Step = store.StepMainMenu
	if err := h.store.Save(ctx, sess); err != nil {
		slog.Error("save session", "phone", phone, "err", err)
		return
	}
	h.sendMainMenu(ctx, phone, sess)
}

// ─────────────────────────────────────────────
// Step 4 – main menu
// ─────────────────────────────────────────────

func (h *Handler) sendMainMenu(ctx context.Context, phone string, sess *store.Session) {
	t := h.t(sess)
	if err := h.wa.SendButtons(ctx, phone, t.Welcome, [][2]string{
		{"action_sos", t.LabelSOS},
		{"action_register", t.LabelRegister},
		{"action_track", t.LabelTrack},
	}); err != nil {
		slog.Error("sendMainMenu", "phone", phone, "err", err)
	}
}

func (h *Handler) handleMainMenuSelection(ctx context.Context, phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	id := buttonID(ir)
	t := h.t(sess)

	var label, handler string
	switch id {
	case "action_sos":
		label, handler = t.LabelSOS, "SOS"
	case "action_register":
		label, handler = t.LabelRegister, "Register"
	case "action_track":
		label, handler = t.LabelTrack, "Track"
	default:
		h.sendMainMenu(ctx, phone, sess)
		return
	}

	// ── TODO: replace with real sub-flow handlers ─────────────────────────────
	msg := fmt.Sprintf(
		"✅ You selected *%s*.\n\n_Nagarsevak: %s | Ward: %s_\n\n_Plug in your %s sub-flow in bot/handler.go._\n\nType anything to return to the main menu.",
		label, sess.NagarsevakName, sess.Ward, handler,
	)
	_ = h.wa.SendText(ctx, phone, msg)
	sess.Step = store.StepMainMenu
	_ = h.store.Save(ctx, sess)
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func (h *Handler) t(sess *store.Session) Strings {
	if t, ok := I18n[sess.Lang]; ok {
		return t
	}
	return I18n["en"]
}

func buttonID(ir *whatsapp.InteractiveReply) string {
	if ir == nil {
		return ""
	}
	if ir.ButtonReply != nil {
		return ir.ButtonReply.ID
	}
	if ir.ListReply != nil {
		return ir.ListReply.ID
	}
	return ""
}

func listID(ir *whatsapp.InteractiveReply) string {
	if ir == nil || ir.ListReply == nil {
		return ""
	}
	return ir.ListReply.ID
}