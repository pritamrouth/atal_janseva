// Package whatsapp wraps the Meta WhatsApp Cloud API v20.
// The HTTP client is tuned for high concurrency:
//   - persistent connection pool (100 idle per host)
//   - exponential backoff retry on 5xx / timeout (up to 3 attempts)
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const apiBase = "https://graph.facebook.com/v20.0"

// Client is a high-concurrency WhatsApp Cloud API client.
type Client struct {
	phoneNumberID string
	accessToken   string
	http          *http.Client
}

// New returns a Client with a tuned HTTP transport.
func New(phoneNumberID, accessToken string) *Client {
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	return &Client{
		phoneNumberID: phoneNumberID,
		accessToken:   accessToken,
		http: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

// ─────────────────────────────────────────────
// Payload types – text
// ─────────────────────────────────────────────

type textBody struct {
	MessagingProduct string  `json:"messaging_product"`
	To               string  `json:"to"`
	Type             string  `json:"type"`
	Text             textObj `json:"text"`
}
type textObj struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

// ─────────────────────────────────────────────
// Payload types – image
// ─────────────────────────────────────────────

type imageBody struct {
	MessagingProduct string   `json:"messaging_product"`
	To               string   `json:"to"`
	Type             string   `json:"type"`
	Image            imageObj `json:"image"`
}
type imageObj struct {
	Link    string `json:"link"`
	Caption string `json:"caption,omitempty"`
}

// ─────────────────────────────────────────────
// Payload types – interactive (buttons / list)
// ─────────────────────────────────────────────

type interactiveBody struct {
	MessagingProduct string         `json:"messaging_product"`
	To               string         `json:"to"`
	Type             string         `json:"type"`
	Interactive      interactiveObj `json:"interactive"`
}
type interactiveObj struct {
	Type   string          `json:"type"`
	Header *interactHeader `json:"header,omitempty"`
	Body   interactText    `json:"body"`
	Footer *interactText   `json:"footer,omitempty"`
	Action interactAction  `json:"action"`
}
type interactHeader struct {
	Type  string    `json:"type"`
	Text  string    `json:"text,omitempty"`
	Image *imageObj `json:"image,omitempty"`
}
type interactText struct{ Text string `json:"text"` }
type interactAction struct {
	Buttons    []interactButton `json:"buttons,omitempty"`
	ButtonText string           `json:"button,omitempty"`
	Sections   []listSection    `json:"sections,omitempty"`
}
type interactButton struct {
	Type  string      `json:"type"`
	Reply buttonReply `json:"reply"`
}
type buttonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
type listSection struct {
	Title string    `json:"title"`
	Rows  []listRow `json:"rows"`
}
type listRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ─────────────────────────────────────────────
// Payload types – CTA URL buttons
// Meta docs: interactive type = "cta_url"
// Each button opens a URL in the device browser.
// ─────────────────────────────────────────────

// CTAButton represents one URL-redirect button.
type CTAButton struct {
	Title string // button label (max 20 chars)
	URL   string // destination URL
}

// ctaActionButton is the action.buttons entry for URL redirect buttons.
type ctaActionButton struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ctaBody is the full payload for a CTA interactive message.
// Meta uses a different structure than reply-buttons for URL buttons.
type ctaBody struct {
	MessagingProduct string    `json:"messaging_product"`
	To               string    `json:"to"`
	Type             string    `json:"type"`
	Interactive      ctaInter  `json:"interactive"`
}
type ctaInter struct {
	Type   string          `json:"type"` // "cta_url"
	Header *interactHeader `json:"header,omitempty"`
	Body   interactText    `json:"body"`
	Footer *interactText   `json:"footer,omitempty"`
	Action ctaInterAction  `json:"action"`
}
type ctaInterAction struct {
	Buttons []ctaActionButton `json:"buttons"`
}

// ListSection is the exported type used when calling SendList.
type ListSection struct {
	Title string
	Rows  [][3]string // [id, title, description]
}

// ─────────────────────────────────────────────
// Public send methods
// ─────────────────────────────────────────────

// SendText sends a plain text message.
func (c *Client) SendText(ctx context.Context, to, text string) error {
	return c.post(ctx, textBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text:             textObj{Body: text},
	})
}

// SendImage sends an image by public URL with an optional caption.
func (c *Client) SendImage(ctx context.Context, to, imageURL, caption string) error {
	return c.post(ctx, imageBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "image",
		Image: imageObj{
			Link:    imageURL,
			Caption: caption,
		},
	})
}

// SendButtons sends up to 3 quick-reply buttons (no URL redirect).
func (c *Client) SendButtons(ctx context.Context, to, bodyText string, buttons [][2]string) error {
	return c.SendButtonsWithImageHeader(ctx, to, bodyText, "", buttons)
}

// SendButtonsWithImageHeader sends buttons with an optional image header.
func (c *Client) SendButtonsWithImageHeader(ctx context.Context, to, bodyText, imageURL string, buttons [][2]string) error {
	btns := make([]interactButton, 0, len(buttons))
	for _, b := range buttons {
		btns = append(btns, interactButton{
			Type:  "reply",
			Reply: buttonReply{ID: b[0], Title: truncate(b[1], 20)},
		})
	}
	msg := interactiveBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: interactiveObj{
			Type:   "button",
			Body:   interactText{Text: bodyText},
			Action: interactAction{Buttons: btns},
		},
	}
	if imageURL != "" {
		msg.Interactive.Header = &interactHeader{Type: "image", Image: &imageObj{Link: imageURL}}
	}
	return c.post(ctx, msg)
}

// SendCTAButtons sends up to 3 URL-redirect buttons using the cta_url
// interactive type. Each button opens its URL in the user's browser.
// buttons is a slice of CTAButton{Title, URL}.
// header is optional (pass "" to omit).
// footer is optional (pass "" to omit).
func (c *Client) SendCTAButtons(ctx context.Context, to, bodyText, header, footer string, buttons []CTAButton) error {
	ctaBtns := make([]ctaActionButton, 0, len(buttons))
	for _, b := range buttons {
		ctaBtns = append(ctaBtns, ctaActionButton{
			Type:  "url",
			URL:   b.URL,
			Title: truncate(b.Title, 20),
		})
	}

	msg := ctaBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: ctaInter{
			Type:   "button",
			Body:   interactText{Text: bodyText},
			Action: ctaInterAction{Buttons: ctaBtns},
		},
	}
	if header != "" {
		msg.Interactive.Header = &interactHeader{Type: "text", Text: header}
	}
	if footer != "" {
		msg.Interactive.Footer = &interactText{Text: footer}
	}
	return c.post(ctx, msg)
}

// SendList sends a list-picker interactive message.
func (c *Client) SendList(ctx context.Context, to, bodyText, buttonLabel string, sections []ListSection) error {
	waSections := make([]listSection, 0, len(sections))
	for _, s := range sections {
		rows := make([]listRow, 0, len(s.Rows))
		for _, r := range s.Rows {
			rows = append(rows, listRow{ID: r[0], Title: truncate(r[1], 24), Description: r[2]})
		}
		waSections = append(waSections, listSection{Title: s.Title, Rows: rows})
	}
	return c.post(ctx, interactiveBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: interactiveObj{
			Type: "list",
			Body: interactText{Text: bodyText},
			Action: interactAction{
				ButtonText: buttonLabel,
				Sections:   waSections,
			},
		},
	})
}

// ─────────────────────────────────────────────
// Internal – post with exponential backoff retry
// ─────────────────────────────────────────────

const maxRetries = 3

var retryBackoff = [maxRetries]time.Duration{
	200 * time.Millisecond,
	600 * time.Millisecond,
	1500 * time.Millisecond,
}

func (c *Client) post(ctx context.Context, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	url := fmt.Sprintf("%s/%s/messages", apiBase, c.phoneNumberID)

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBackoff[attempt-1]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			slog.Warn("WA API request failed, retrying", "attempt", attempt+1, "err", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 500 {
			if resp.StatusCode >= 400 {
				return fmt.Errorf("WA API %d: %s", resp.StatusCode, string(body))
			}
			return nil
		}

		lastErr = fmt.Errorf("attempt %d: WA API %d: %s", attempt+1, resp.StatusCode, string(body))
		slog.Warn("WA API 5xx, retrying", "attempt", attempt+1, "status", resp.StatusCode)
	}
	return fmt.Errorf("all %d attempts failed: %w", maxRetries, lastErr)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}