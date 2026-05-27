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

// ataljansevaDomain is the base URL used to build all redirect links.
const ataljansevaDomain = "https://ataljanseva.in"

// ataljansevaLogoURL is the publicly hosted Ataljanseva logo.
// For development: use ngrok URL like https://abc123.ngrok.io/public/Ataljanseva_Without_WebPortal.png
// For production: use https://ataljanseva.in/public/Ataljanseva_Without_WebPortal.png
const ataljansevaLogoURL = "https://res.cloudinary.com/dkgfw2zf0/image/upload/v1779864154/AtalJanseva_512x512_rtjjqj.png"

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

	// Reset inactivity clock on every inbound message
	h.inactive.Touch(ctx, phone)

	sess, err := h.store.Get(ctx, phone)
	if err != nil {
		log.Error("get session", "err", err)
		return
	}

	// Global reset command
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
			// Send PIN prompt with image after language selection
			h.sendPinPromptWithImage(ctx, phone, sess)
		} else {
			h.sendLanguagePicker(ctx, phone, phone)
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

// sendLanguagePicker sends the Ataljanseva logo (TASK 1) followed immediately
// by the language-picker button message with the user's real phone number.
func (h *Handler) sendLanguagePicker(ctx context.Context, phone, rawPhone string) {
	greeting := GreetingFor("en", rawPhone)
	if err := h.wa.SendButtonsWithImageHeader(ctx, phone, greeting, ataljansevaLogoURL, [][2]string{
		{"lang_en", "English"},
		{"lang_mr", "मराठी"},
		{"lang_hi", "हिंदी"},
	}); err != nil {
		slog.Error("sendLanguagePicker", "phone", phone, "err", err)
	}
}

// sendPinPromptWithImage sends the PIN prompt with an image header
func (h *Handler) sendPinPromptWithImage(ctx context.Context, phone string, sess *store.Session) {
	if err := h.wa.SendImage(ctx, phone, ataljansevaLogoURL, h.t(sess).PinPrompt); err != nil {
		slog.Error("sendPinPromptWithImage", "phone", phone, "err", err)
		// Fallback to text if image fails
		_ = h.wa.SendText(ctx, phone, h.t(sess).PinPrompt)
	}
}

// ─────────────────────────────────────────────
// Step 1 – PIN → DB
// ─────────────────────────────────────────────

func (h *Handler) handlePin(ctx context.Context, phone string, sess *store.Session, raw string) {
	pin, wardHint := parseUserInput(raw)
	asciiPin := normalizePin(pin)

	if len([]rune(asciiPin)) != 6 {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}

	// Ward is now mandatory — must be provided with pincode in one message
	if wardHint == "" {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}

	// Fetch location and wards in parallel
	var loc *db.LocationInfo
	var wards []db.Ward
	var locErr, wardsErr error

	// Use goroutines to fetch location and wards concurrently
	locChan := make(chan *db.LocationInfo, 1)
	locErrChan := make(chan error, 1)
	wardsChan := make(chan []db.Ward, 1)
	wardsErrChan := make(chan error, 1)

	// Fetch location (queries both ASCII and Devanagari columns)
	go func() {
		l, err := h.repo.LocationByPincode(ctx, asciiPin)
		locChan <- l
		locErrChan <- err
	}()

	// Fetch wards (queries both ASCII and Devanagari columns)
	go func() {
		w, err := h.repo.WardsByPincode(ctx, asciiPin)
		wardsChan <- w
		wardsErrChan <- err
	}()

	// Wait for both to complete
	loc = <-locChan
	locErr = <-locErrChan
	wards = <-wardsChan
	wardsErr = <-wardsErrChan

	// Always store normalized ASCII pincode
	sess.Pincode = asciiPin

	// Handle location errors
	if locErr == sql.ErrNoRows {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}

	if locErr != nil {
		slog.Error("LocationByPincode", "pin", pin, "lang", sess.Lang, "err", locErr)
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}

	sess.State         = loc.State
	sess.District      = loc.District
	sess.StateHindi    = loc.StateHindi
	sess.DistrictHindi = loc.DistrictHindi

	// Handle wards errors
	if wardsErr != nil || len(wards) == 0 {
		slog.Error("WardsByPincode in handlePin", "pin", sess.Pincode, "err", wardsErr)
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}

	// Match ward from user input
	matched := ""
	matchedHindi := ""
	for _, w := range wards {
		if wardMatchesHint(w.Code, wardHint) || wardMatchesHint(w.CodeHindi, wardHint) {
			matched = w.Code
			matchedHindi = w.CodeHindi
			break
		}
	}
	if matched == "" {
		h.sendInvalidWardHint(ctx, phone, sess, wardHint, wards)
		return
	}

	// Valid PIN + ward — proceed directly to nagarsevak selection
	sess.Ward = matched
	sess.WardHindi = matchedHindi
	sess.Step = store.StepNagarsevak
	if err := h.store.Save(ctx, sess); err != nil {
		slog.Error("save session", "phone", phone, "err", err)
		return
	}
	h.promptNagarsevak(ctx, phone, sess)
}

// sendInvalidWardHint sends the basic InvalidPin error message without extras
func (h *Handler) sendInvalidWardHint(ctx context.Context, phone string, sess *store.Session, hint string, wards []db.Ward) {
	_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
}

// ─────────────────────────────────────────────
// Step 2 – ward list
// ─────────────────────────────────────────────

func (h *Handler) promptWard(ctx context.Context, phone string, sess *store.Session) {
	var wards []db.Ward
	// Query searches both ASCII and Devanagari columns
	wards, err := h.repo.WardsByPincode(ctx, sess.Pincode)
	if err != nil {
		slog.Error("WardsByPincode", "pin", sess.Pincode, "err", err)
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}
	if len(wards) == 0 {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		sess.Step = store.StepLangChosen
		_ = h.store.Save(ctx, sess)
		return
	}

	t := h.t(sess)

	// Pick localised state/district label
	stateName    := sess.State
	districtName := sess.District
	if (sess.Lang == "mr" || sess.Lang == "hi") && sess.StateHindi != "" {
		stateName = sess.StateHindi
	}
	if (sess.Lang == "mr" || sess.Lang == "hi") && sess.DistrictHindi != "" {
		districtName = sess.DistrictHindi
	}

	bodyText := fmt.Sprintf(t.WardPrompt, stateName, districtName)
	_ = h.wa.SendText(ctx, phone, bodyText)
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
// Step 3 – nagarsevak list + profile photos
// ─────────────────────────────────────────────

func (h *Handler) promptNagarsevak(ctx context.Context, phone string, sess *store.Session) {
	var nagarsevaks []db.Nagarsevak
	// Query searches both ASCII and Devanagari columns
	nagarsevaks, err := h.repo.NagarsevaksByWard(ctx, sess.Pincode, sess.Ward)
	
	if err != nil {
        slog.Error("NagarsevaksByWard", "pin", sess.Pincode, "ward", sess.Ward, "err", err)
        _ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
        return
    }
    if len(nagarsevaks) == 0 {
        _ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
        sess.Step = store.StepWardChosen
        _ = h.store.Save(ctx, sess)
        h.promptWard(ctx, phone, sess)
        return
    }

    t := h.t(sess)

    // Build list rows from nagarsevaks
    rows := make([][3]string, 0, len(nagarsevaks))
	for _, ns := range nagarsevaks {
		displayName := ns.FullName
		if (sess.Lang == "mr" || sess.Lang == "hi") && ns.NameHindi != "" {
			displayName = ns.NameHindi
		}
		rows = append(rows, [3]string{
			"ns_" + ns.ID,        // ID (for callback)
			displayName,           // Name
			ns.Party,              // Party/Description
		})
	}

	// Use Hindi ward name if available and language is Hindi/Marathi
	displayWard := sess.Ward
	if (sess.Lang == "mr" || sess.Lang == "hi") && sess.WardHindi != "" {
		displayWard = sess.WardHindi
	}

	bodyText := fmt.Sprintf(t.WardPrompt, sess.Pincode, displayWard)

	if err := h.wa.SendList(ctx, phone, bodyText, "🏅 Select Corporator", []whatsapp.ListSection{
		{Title: "👥 Corporators", Rows: rows},
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
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		h.promptNagarsevak(ctx, phone, sess)
		return
	}
	sess.NagarsevakID   = ns.ID
	sess.NagarsevakName = ns.FullName
	sess.NagarsevakSlug = ns.Slug // TASK 2 – persist slug for URL generation
	sess.Step           = store.StepMainMenu
	if err := h.store.Save(ctx, sess); err != nil {
		slog.Error("save session", "phone", phone, "err", err)
		return
	}
	h.sendMainMenu(ctx, phone, sess)
}

// ─────────────────────────────────────────────
// Step 4 – main menu  (TASK 2: CTA URL buttons)
// ─────────────────────────────────────────────

// nagarsevakURL builds a full redirect URL from the slug and path suffix.
// e.g. slug="vinodmishra", suffix="sos" → "https://ataljanseva.in/vinodmishra/sos"
func nagarsevakURL(slug, suffix string) string {
	return fmt.Sprintf("%s/%s/%s", ataljansevaDomain, slug, suffix)
}

func (h *Handler) sendMainMenu(ctx context.Context, phone string, sess *store.Session) {
	t := h.t(sess)

	ns, err := h.repo.NagarsevakByID(ctx, sess.NagarsevakID)
	if err != nil {
		slog.Error("sendMainMenu: NagarsevakByID", "phone", phone, "err", err)
		ns = &db.Nagarsevak{
			FullName:     sess.NagarsevakName,
			ProfilePhoto: "",
		}
	}

	// Inject nagarsevak slug into Welcome message
	welcomeMsg := strings.ReplaceAll(t.Welcome, "{{slug}}", sess.NagarsevakSlug)

	if ns.ProfilePhoto != "" {
		// Single message: photo + full welcome text as caption
		caption := fmt.Sprintf("🏅 *%s*\n🎖 %s  ·  🏙 Ward %s\n\n%s",
			ns.FullName, ns.Party, ns.Ward, welcomeMsg)
		if err := h.wa.SendImage(ctx, phone, ns.ProfilePhoto, caption); err != nil {
			slog.Warn("sendMainMenu: profile photo send failed", "phone", phone, "err", err)
			// fallback to plain text if image fails
			_ = h.wa.SendText(ctx, phone, welcomeMsg)
		}
	} else {
		// No photo — send nagarsevak details + welcome as plain text
		header := fmt.Sprintf("🏅 *%s*\n🎖 %s  ·  🏙 Ward %s\n\n", ns.FullName, ns.Party, ns.Ward)
		_ = h.wa.SendText(ctx, phone, header+welcomeMsg)
	}

	// CTA URL buttons - send each with its own language-based header
	buttons := []struct {
		header string
		button whatsapp.CTAButton
	}{
		{t.SOSHeader, whatsapp.CTAButton{Title: t.LabelSOS, URL: nagarsevakURL(sess.NagarsevakSlug, "sos")}},
		{t.ComplaintHeader, whatsapp.CTAButton{Title: t.LabelRegister, URL: nagarsevakURL(sess.NagarsevakSlug, "grievance")}},
		{t.TrackHeader, whatsapp.CTAButton{Title: t.LabelTrack, URL: nagarsevakURL(sess.NagarsevakSlug, "track-issue")}},
	}

	for _, b := range buttons {
		if err := h.wa.SendCTAButtons(ctx, phone, b.header, "", "", []whatsapp.CTAButton{b.button}); err != nil {
			slog.Error("sendMainMenu SendCTAButtons", "phone", phone, "err", err)
		}
	}
}

// handleMainMenuSelection handles cases where the user somehow sends an
// interactive reply at StepMainMenu (shouldn't happen with CTA buttons, but
// guard against it gracefully).
func (h *Handler) handleMainMenuSelection(ctx context.Context, phone string, sess *store.Session, ir *whatsapp.InteractiveReply) {
	// CTA URL buttons don't send a reply back to the server — the user is
	// redirected to the URL in their browser. If we somehow receive an
	// interactive event here (e.g. from an older cached message), just
	// re-display the main menu.
	h.sendMainMenu(ctx, phone, sess)
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

func normalizePin(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if r >= '०' && r <= '९' {
			b.WriteRune('0' + (r - '०'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseUserInput splits on comma or space.
// "400601"        → pin="400601", wardHint=""
// "400601, TES1"  → pin="400601", wardHint="TES1"
// "400601,TES1"   → pin="400601", wardHint="TES1"
// "400601 TES1"   → pin="400601", wardHint="TES1"
func parseUserInput(raw string) (pin string, wardHint string) {
	// Try comma first, then fall back to space
	var parts []string
	if strings.Contains(raw, ",") {
		parts = strings.SplitN(raw, ",", 2)
	} else {
		parts = strings.SplitN(raw, " ", 2)
	}
	pin = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		wardHint = strings.TrimSpace(parts[1])
	}
	return
}

// wardMatchesHint does exact case-insensitive match only.
func wardMatchesHint(wardCode, hint string) bool {
	if hint == "" {
		return false
	}
	return strings.EqualFold(wardCode, hint)
}

