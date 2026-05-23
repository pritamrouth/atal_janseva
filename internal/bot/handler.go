// Package bot implements the Ataljanseva onboarding conversation flow
// driven by live data from PostgreSQL.
package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/ataljanseva/whatsapp-bot/internal/db"
	"github.com/ataljanseva/whatsapp-bot/internal/store"
	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
)

// Handler processes inbound WhatsApp messages and drives the flow forward.
type Handler struct {
	wa    *whatsapp.Client
	store *store.Store
	repo  *db.Repo
}

// New returns a Handler wired to a WhatsApp client, session store, and DB repo.
func New(wa *whatsapp.Client, s *store.Store, repo *db.Repo) *Handler {
	return &Handler{wa: wa, store: s, repo: repo}
}

// ─────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────

// HandleRaw dispatches a single inbound message through the state machine.
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
			id := buttonID(msg.Interactive)
			switch id {
			case "lang_en":
				sess.Lang = "en"
			case "lang_mr":
				sess.Lang = "mr"
			case "lang_hi":
				sess.Lang = "hi"
			default:
				h.sendLanguagePicker(phone)
				return
			}
			sess.Step = store.StepLangChosen
			h.store.Save(sess)
			_ = h.wa.SendText(phone, h.t(sess).PinPrompt)
		} else {
			h.sendLanguagePicker(phone)
		}

	// ── STEP 1: waiting for PIN ───────────────────────────
	case store.StepLangChosen:
		if msg.Type == "text" {
			h.handlePin(phone, sess, strings.TrimSpace(msg.Text.Body))
		} else {
			_ = h.wa.SendText(phone, h.t(sess).PinPrompt)
		}

	// ── STEP 2: ward selection ────────────────────────────
	case store.StepWardChosen:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleWardReply(phone, sess, msg.Interactive)
		} else {
			h.promptWard(phone, sess)
		}

	// ── STEP 3: nagarsevak selection ─────────────────────
	case store.StepNagarsevak:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleNagarsevakReply(phone, sess, msg.Interactive)
		} else {
			h.promptNagarsevak(phone, sess)
		}

	// ── STEP 4: main menu ────────────────────────────────
	case store.StepMainMenu:
		if msg.Type == "interactive" && msg.Interactive != nil {
			h.handleMainMenuSelection(phone, sess, msg.Interactive)
		} else {
			h.sendMainMenu(phone, sess)
		}
	}
}

// ─────────────────────────────────────────────
// Step 0 – language picker
// ─────────────────────────────────────────────

func (h *Handler) sendLanguagePicker(phone string) {
	err := h.wa.SendButtons(phone, I18n["en"].Greeting, [][2]string{
		{"lang_en", "🇬🇧 English"},
		{"lang_mr", "🇮🇳 मराठी"},
		{"lang_hi", "🇮🇳 हिंदी"},
	})
	if err != nil {
		log.Printf("[bot] sendLanguagePicker %s: %v", phone, err)
	}
}

// ─────────────────────────────────────────────
// Step 1 – PIN → DB lookup
// ─────────────────────────────────────────────

func (h *Handler) handlePin(phone string, sess *store.Session, pin string) {
	// Validate: must be 6 digits
	if len(pin) != 6 {
		_ = h.wa.SendText(phone, h.t(sess).InvalidPin)
		return
	}

	loc, err := h.repo.LocationByPincode(pin)
	if err == sql.ErrNoRows {
		_ = h.wa.SendText(phone, h.t(sess).InvalidPin)
		return
	}
	if err != nil {
		log.Printf("[bot] LocationByPincode %s: %v", pin, err)
		_ = h.wa.SendText(phone, "⚠️ A database error occurred. Please try again shortly.")
		return
	}

	sess.Pincode = pin
	sess.State = loc.State
	sess.District = loc.District
	sess.Step = store.StepWardChosen
	h.store.Save(sess)
	h.promptWard(phone, sess)
}

// ─────────────────────────────────────────────
// Step 2 – ward list from DB
// ─────────────────────────────────────────────

func (h *Handler) promptWard(phone string, sess *store.Session) {
	wards, err := h.repo.WardsByPincode(sess.Pincode)
	if err != nil {
		log.Printf("[bot] WardsByPincode %s: %v", sess.Pincode, err)
		_ = h.wa.SendText(phone, "⚠️ Could not load ward data. Please try again.")
		return
	}
	if len(wards) == 0 {
		_ = h.wa.SendText(phone, "⚠️ No wards found for this PIN code. Please check and re-enter.")
		// Roll back to PIN prompt
		sess.Step = store.StepLangChosen
		h.store.Save(sess)
		return
	}

	t := h.t(sess)

	// Localise state / district label
	stateLabel, districtLabel := sess.State, sess.District

	rows := make([][3]string, 0, len(wards))
	for _, w := range wards {
		label := w.Code
		if sess.Lang != "en" && w.CodeHindi != "" {
			label = w.Code + " (" + w.CodeHindi + ")"
		}
		rows = append(rows, [3]string{
			"ward_" + w.Code, // id — ward code is unique per pincode
			label,
			"",
		})
	}

	bodyText := fmt.Sprintf(t.WardPrompt, stateLabel, districtLabel)
	err = h.wa.SendList(phone, bodyText, "Select Ward", []whatsapp.ListSection{
		{Title: "Available Wards", Rows: rows},
	})
	if err != nil {
		log.Printf("[bot] promptWard SendList %s: %v", phone, err)
	}
}

func (h *Handler) handleWardReply(phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	id := listID(ir)
	if !strings.HasPrefix(id, "ward_") {
		h.promptWard(phone, sess)
		return
	}
	wardCode := strings.TrimPrefix(id, "ward_")
	sess.Ward = wardCode
	sess.Step = store.StepNagarsevak
	h.store.Save(sess)
	h.promptNagarsevak(phone, sess)
}

// ─────────────────────────────────────────────
// Step 3 – nagarsevak list from DB
// ─────────────────────────────────────────────

func (h *Handler) promptNagarsevak(phone string, sess *store.Session) {
	nagarsevaks, err := h.repo.NagarsevaksByWard(sess.Pincode, sess.Ward)
	if err != nil {
		log.Printf("[bot] NagarsevaksByWard pin=%s ward=%s: %v", sess.Pincode, sess.Ward, err)
		_ = h.wa.SendText(phone, "⚠️ Could not load nagarsevak data. Please try again.")
		return
	}
	if len(nagarsevaks) == 0 {
		_ = h.wa.SendText(phone, "⚠️ No nagarsevaks found for Ward "+sess.Ward+". Please select a different ward.")
		sess.Step = store.StepWardChosen
		h.store.Save(sess)
		h.promptWard(phone, sess)
		return
	}

	t := h.t(sess)
	rows := make([][3]string, 0, len(nagarsevaks))
	for _, ns := range nagarsevaks {
		displayName := ns.FullName
		if sess.Lang != "en" && ns.NameHindi != "" {
			displayName = ns.NameHindi
		}
		rows = append(rows, [3]string{
			"ns_" + ns.ID,
			displayName,
			ns.Party + " · Ward " + ns.Ward,
		})
	}

	err = h.wa.SendList(phone, t.NagarsevakPrompt, "Select Nagarsevak", []whatsapp.ListSection{
		{Title: "Nagarsevaks", Rows: rows},
	})
	if err != nil {
		log.Printf("[bot] promptNagarsevak SendList %s: %v", phone, err)
	}
}

func (h *Handler) handleNagarsevakReply(phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	id := listID(ir)
	if !strings.HasPrefix(id, "ns_") {
		h.promptNagarsevak(phone, sess)
		return
	}
	nsID := strings.TrimPrefix(id, "ns_")

	ns, err := h.repo.NagarsevakByID(nsID)
	if err != nil {
		log.Printf("[bot] NagarsevakByID %s: %v", nsID, err)
		_ = h.wa.SendText(phone, "⚠️ Could not find the selected nagarsevak. Please try again.")
		h.promptNagarsevak(phone, sess)
		return
	}

	sess.NagarsevakID = ns.ID
	sess.NagarsevakName = ns.FullName
	sess.Step = store.StepMainMenu
	h.store.Save(sess)
	h.sendMainMenu(phone, sess)
}

// ─────────────────────────────────────────────
// Step 4 – main menu
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

	var label, handler string
	switch id {
	case "action_sos":
		label, handler = t.LabelSOS, "SOS"
	case "action_register":
		label, handler = t.LabelRegister, "Register"
	case "action_track":
		label, handler = t.LabelTrack, "Track"
	default:
		h.sendMainMenu(phone, sess)
		return
	}

	// ── TODO: replace with real sub-flow handlers ──────────
	confirmation := fmt.Sprintf(
		"✅ You selected *%s*.\n\n_Nagarsevak: %s_\n_Ward: %s_\n\n_Plug in your %s sub-flow handler in bot/handler.go._\n\nType anything to return to the main menu.",
		label, sess.NagarsevakName, sess.Ward, handler,
	)
	_ = h.wa.SendText(phone, confirmation)
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
	if t, ok := I18n[lang]; ok {
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
