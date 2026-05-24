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
// Served from your own CDN / storage — replace if the URL changes.
const ataljansevaLogoURL = "https://ataljanseva.in/logo-1.png"

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
			_ = h.wa.SendText(ctx, phone, h.t(sess).PinPrompt)
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
		{"lang_en", "🇮🇳 English"},
		{"lang_mr", "🇮🇳 मराठी"},
		{"lang_hi", "🇮🇳 हिंदी"},
	}); err != nil {
		slog.Error("sendLanguagePicker", "phone", phone, "err", err)
	}
}

// ─────────────────────────────────────────────
// Step 1 – PIN → DB
// ─────────────────────────────────────────────

func (h *Handler) handlePin(ctx context.Context, phone string, sess *store.Session, pin string) {
	pin = strings.TrimSpace(pin)

	// Normalize to ASCII only for length validation
	asciiPin := normalizePin(pin)
	if len([]rune(asciiPin)) != 6 {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}

	var loc *db.LocationInfo
	var err error
	if sess.Lang == "mr" || sess.Lang == "hi" {
		// Query pincode_hindi column using the original Devanagari input
		loc, err = h.repo.LocationByPincodeHindi(ctx, pin)
		// Store the ASCII version in session for downstream ward/nagarsevak queries
		sess.Pincode = pin // keep Devanagari — all Hindi queries use pincode_hindi
	} else {
		loc, err = h.repo.LocationByPincode(ctx, asciiPin)
		sess.Pincode = asciiPin
	}

	if err == sql.ErrNoRows {
		_ = h.wa.SendText(ctx, phone, h.t(sess).InvalidPin)
		return
	}
	if err != nil {
		slog.Error("LocationByPincode", "pin", pin, "lang", sess.Lang, "err", err)
		_ = h.wa.SendText(ctx, phone, "⚠️ Database error, please try again shortly.")
		return
	}

	sess.State = loc.State
	sess.District = loc.District

	sess.StateHindi    = loc.StateHindi
	sess.DistrictHindi = loc.DistrictHindi

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
	var wards []db.Ward
	var err error
	if sess.Lang == "mr" || sess.Lang == "hi" {
		wards, err = h.repo.WardsByPincodeHindi(ctx, sess.Pincode)
	} else {
		wards, err = h.repo.WardsByPincode(ctx, sess.Pincode)
	}
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

	// Pick localised state/district label
	stateName    := sess.State
	districtName := sess.District
	if (sess.Lang == "mr" || sess.Lang == "hi") && sess.StateHindi != "" {
		stateName = sess.StateHindi
	}
	if (sess.Lang == "mr" || sess.Lang == "hi") && sess.DistrictHindi != "" {
		districtName = sess.DistrictHindi
	}

	rows := make([][3]string, 0, len(wards))
	for _, w := range wards {
		label := w.Code
		if (sess.Lang == "mr" || sess.Lang == "hi") && w.CodeHindi != "" {
			label = w.CodeHindi  // ← show ward_hindi label in list
		}
		rows = append(rows, [3]string{"ward_" + w.Code, label, ""})
	}

	bodyText := fmt.Sprintf(t.WardPrompt, stateName, districtName)
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
// Step 3 – nagarsevak list + profile photos
// ─────────────────────────────────────────────

func (h *Handler) promptNagarsevak(ctx context.Context, phone string, sess *store.Session) {
	var nagarsevaks []db.Nagarsevak
	var err error
	if sess.Lang == "mr" || sess.Lang == "hi" {
		nagarsevaks, err = h.repo.NagarsevaksByWardHindi(ctx, sess.Pincode, sess.Ward)
	} else {
		nagarsevaks, err = h.repo.NagarsevaksByWard(ctx, sess.Pincode, sess.Ward)
	}
	
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

    // ✅ No photo loop here anymore — photo moves to sendMainMenu after selection

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

    // 1. If photo exists, send it with nagarsevak details as caption
    //    This appears at the top of the "card" the user sees.
    if ns.ProfilePhoto != "" {
        caption := fmt.Sprintf("🏅 *%s*\n🎖 %s  ·  🏙 Ward %s",
            ns.FullName, ns.Party, ns.Ward)
        if err := h.wa.SendImage(ctx, phone, ns.ProfilePhoto, caption); err != nil {
            slog.Warn("sendMainMenu: profile photo send failed", "phone", phone, "err", err)
        }
    }

    // 2. Welcome message body
    footer := fmt.Sprintf("📋 Nagarsevak: %s", sess.NagarsevakName)
    if err := h.wa.SendText(ctx, phone, t.Welcome); err != nil {
        slog.Error("sendMainMenu: SendText", "phone", phone, "err", err)
    }

    // 3. CTA URL action buttons
    ctaButtons := []whatsapp.CTAButton{
        {Title: t.LabelSOS,      URL: nagarsevakURL(sess.NagarsevakSlug, "sos")},
        {Title: t.LabelRegister, URL: nagarsevakURL(sess.NagarsevakSlug, "grievance")},
        {Title: t.LabelTrack,    URL: nagarsevakURL(sess.NagarsevakSlug, "track-issue")},
    }
    if err := h.wa.SendCTAButtons(ctx, phone, t.LabelSOS, "", footer, ctaButtons); err != nil {
        slog.Error("sendMainMenu SendCTAButtons", "phone", phone, "err", err)
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