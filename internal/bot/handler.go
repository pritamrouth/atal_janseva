// Package bot implements the Ataljanseva onboarding conversation flow.
package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/ataljanseva/whatsapp-bot/internal/store"
	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
)

// Handler processes inbound WhatsApp messages and drives the flow forward.
type Handler struct {
	wa    *whatsapp.Client
	store *store.Store
}

// New returns a Handler wired to a WhatsApp client and session store.
func New(wa *whatsapp.Client, s *store.Store) *Handler {
	return &Handler{wa: wa, store: s}
}

// ─────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────

// Handle dispatches a single inbound message.
func (h *Handler) Handle(msg whatsapp.Message) {
	phone := msg.From
	sess := h.store.Get(phone)

	// Global reset command
	if msg.Type == "text" && strings.EqualFold(strings.TrimSpace(msg.Text.Body), "reset") {
		h.store.Reset(phone)
		_ = h.wa.SendText(phone, "🔄 Session reset. Type anything to start over.")
		return
	}

	switch sess.Step {

	case store.StepStart:
		// Show language picker (always, even on re-entry after reset)
		h.sendLanguagePicker(phone, sess)

	case store.StepLangChosen:
		// Expecting PIN code text
		if msg.Type == "text" {
			h.handlePin(phone, sess, strings.TrimSpace(msg.Text.Body))
		} else {
			t := h.t(sess)
			_ = h.wa.SendText(phone, t.PinPrompt)
		}

	case store.StepWardChosen:
		// Expecting a list_reply for ward or nagarsevak selection
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleInteractive(phone, sess, msg.Interactive)
		} else {
			h.promptWard(phone, sess)
		}

	case store.StepMainMenu:
		// Expecting button_reply for SOS / Register / Track
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleMainMenuSelection(phone, sess, msg.Interactive)
		} else {
			h.sendMainMenu(phone, sess)
		}
	}
}

// ─────────────────────────────────────────────
// Step 0 → Language picker
// ─────────────────────────────────────────────

func (h *Handler) sendLanguagePicker(phone string, sess *store.Session) {
	// We use English greeting regardless (meta-language)
	greeting := I18n["en"].Greeting

	err := h.wa.SendButtons(phone, greeting, [][2]string{
		{"lang_en", "🇬🇧 English"},
		{"lang_mr", "🇮🇳 मराठी"},
		{"lang_hi", "🇮🇳 हिंदी"},
	})
	if err != nil {
		log.Printf("[bot] sendLanguagePicker %s: %v", phone, err)
	}

	// Move session to StepLangChosen so next input is a button reply
	// We handle the actual language inside handleInteractive
	// But StepStart needs to progress when any button is tapped,
	// so we leave the session at StepStart here and advance inside handleInteractive.
}

// ─────────────────────────────────────────────
// Language selection arrives as button_reply
// We need to intercept it at StepStart too.
// Override Handle for StepStart interactive messages.
// ─────────────────────────────────────────────

func init() {} // satisfy compiler

// We re-route the Handle to also catch button replies when StepStart.
// (Overwrite the StepStart case above to handle interactive too.)

// HandleRaw is the true entry-point; Handle wraps it.
func (h *Handler) HandleRaw(msg whatsapp.Message) {
	phone := msg.From
	sess := h.store.Get(phone)

	// Global reset
	if msg.Type == "text" && strings.EqualFold(strings.TrimSpace(msg.Text.Body), "reset") {
		h.store.Reset(phone)
		_ = h.wa.SendText(phone, "🔄 Session reset. Type anything to start over.")
		return
	}

	switch sess.Step {

	// ── STEP 0: show language picker ──────────────────────
	case store.StepStart:
		if msg.Type == "interactive" && msg.Interactive != nil {
			// User tapped a language button
			id := buttonID(msg.Interactive)
			switch id {
			case "lang_en":
				sess.Lang = "en"
			case "lang_mr":
				sess.Lang = "mr"
			case "lang_hi":
				sess.Lang = "hi"
			default:
				h.sendLanguagePicker(phone, sess)
				return
			}
			sess.Step = store.StepLangChosen
			h.store.Save(sess)
			t := h.t(sess)
			_ = h.wa.SendText(phone, t.PinPrompt)
		} else {
			// First contact or any text – show language picker
			h.sendLanguagePicker(phone, sess)
		}

	// ── STEP 1: waiting for PIN ───────────────────────────
	case store.StepLangChosen:
		if msg.Type == "text" {
			h.handlePin(phone, sess, strings.TrimSpace(msg.Text.Body))
		} else {
			_ = h.wa.SendText(phone, h.t(sess).PinPrompt)
		}

	// ── STEP 2: ward / nagarsevak selection ──────────────
	case store.StepWardChosen:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleInteractive(phone, sess, msg.Interactive)
		} else {
			h.promptWard(phone, sess)
		}

	// ── STEP 3: main menu ────────────────────────────────
	case store.StepMainMenu:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleMainMenuSelection(phone, sess, msg.Interactive)
		} else {
			h.sendMainMenu(phone, sess)
		}
	}
}

// ─────────────────────────────────────────────
// PIN handling
// ─────────────────────────────────────────────

func (h *Handler) handlePin(phone string, sess *store.Session, pin string) {
	data, ok := Pincodes[pin]
	if !ok {
		_ = h.wa.SendText(phone, h.t(sess).InvalidPin)
		return
	}
	sess.Pincode = pin
	sess.State = data.State
	sess.District = data.District
	sess.Step = store.StepWardChosen
	sess.Pending = "ward_select"
	h.store.Save(sess)
	h.promptWard(phone, sess)
}

func (h *Handler) promptWard(phone string, sess *store.Session) {
	data := Pincodes[sess.Pincode]
	t := h.t(sess)
	bodyText := fmt.Sprintf(t.WardPrompt, data.State, data.District)

	rows := make([][3]string, 0, len(data.Wards))
	for i, w := range data.Wards {
		rows = append(rows, [3]string{
			fmt.Sprintf("ward_%d", i),
			w,
			"",
		})
	}
	err := h.wa.SendList(phone, bodyText, "Select Ward", []whatsapp.ListSection{
		{Title: "Available Wards", Rows: rows},
	})
	if err != nil {
		log.Printf("[bot] promptWard %s: %v", phone, err)
	}
}

// ─────────────────────────────────────────────
// Interactive reply handler (list / button)
// ─────────────────────────────────────────────

func (h *Handler) handleInteractive(phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	switch sess.Pending {

	case "ward_select":
		title := listTitle(ir)
		if title == "" {
			h.promptWard(phone, sess)
			return
		}
		sess.Ward = title
		sess.Pending = "nagarsevak_select"
		h.store.Save(sess)
		h.promptNagarsevak(phone, sess)

	case "nagarsevak_select":
		id := listID(ir)
		if id == "" {
			h.promptNagarsevak(phone, sess)
			return
		}
		// id is "ns_0", "ns_1", …
		nagarsevaks := NagarsevakDB[sess.Ward]
		idx := 0
		fmt.Sscanf(id, "ns_%d", &idx)
		if idx < 0 || idx >= len(nagarsevaks) {
			h.promptNagarsevak(phone, sess)
			return
		}
		sess.Nagarsevak = nagarsevaks[idx].Name
		sess.Step = store.StepMainMenu
		sess.Pending = ""
		h.store.Save(sess)
		h.sendMainMenu(phone, sess)

	default:
		// Shouldn't happen – just re-send the ward picker
		h.promptWard(phone, sess)
	}
}

func (h *Handler) promptNagarsevak(phone string, sess *store.Session) {
	nagarsevaks := NagarsevakDB[sess.Ward]
	t := h.t(sess)

	rows := make([][3]string, 0, len(nagarsevaks))
	for i, ns := range nagarsevaks {
		rows = append(rows, [3]string{
			fmt.Sprintf("ns_%d", i),
			ns.Name,
			ns.Party + " · " + sess.Ward,
		})
	}
	err := h.wa.SendList(phone, t.NagarsevakPrompt, "Select Nagarsevak", []whatsapp.ListSection{
		{Title: "Nagarsevaks", Rows: rows},
	})
	if err != nil {
		log.Printf("[bot] promptNagarsevak %s: %v", phone, err)
	}
}

// ─────────────────────────────────────────────
// Main menu
// ─────────────────────────────────────────────

func (h *Handler) sendMainMenu(phone string, sess *store.Session) {
	t := h.t(sess)
	err := h.wa.SendButtons(phone, t.Welcome, [][2]string{
		{"action_sos", t.LabelSOS},
		{"action_register", t.LabelRegister},
		{"action_track", t.LabelTrack},
	})
	if err != nil {
		log.Printf("[bot] sendMainMenu %s: %v", phone, err)
	}
}

func (h *Handler) handleMainMenuSelection(phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	id := buttonID(ir)
	t := h.t(sess)

	var (
		label   string
		handler string
	)
	switch id {
	case "action_sos":
		label = t.LabelSOS
		handler = "SOS"
	case "action_register":
		label = t.LabelRegister
		handler = "Register"
	case "action_track":
		label = t.LabelTrack
		handler = "Track"
	default:
		h.sendMainMenu(phone, sess)
		return
	}

	_ = label
	// TODO: plug in your SOS / Register / Track sub-flows here.
	// For now we echo confirmation and loop back to the main menu.
	confirmation := fmt.Sprintf(
		"✅ You selected *%s*.\n\n_This is where the %s flow begins. Plug in your sub-flow handler in bot/handler.go._\n\nType anything to return to the main menu.",
		label, handler,
	)
	_ = h.wa.SendText(phone, confirmation)
	// Reset to main menu so next message re-shows options
	sess.Step = store.StepMainMenu
	h.store.Save(sess)
}

// ─────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────

func (h *Handler) t(sess *store.Session) Strings {
	lang := sess.Lang
	if lang == "" {
		lang = "en"
	}
	t, ok := I18n[lang]
	if !ok {
		return I18n["en"]
	}
	return t
}

// buttonID extracts the reply ID from a button_reply or list_reply.
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

// listID extracts the ID from a list_reply.
func listID(ir *whatsapp.InteractiveReply) string {
	if ir == nil || ir.ListReply == nil {
		return ""
	}
	return ir.ListReply.ID
}

// listTitle extracts the title from a list_reply.
func listTitle(ir *whatsapp.InteractiveReply) string {
	if ir == nil || ir.ListReply == nil {
		return ""
	}
	return ir.ListReply.Title
}
