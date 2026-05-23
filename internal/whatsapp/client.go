// Package whatsapp wraps the Meta WhatsApp Cloud API v20.
package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiBase = "https://graph.facebook.com/v20.0"

// Client is a thin wrapper around the WhatsApp Cloud API.
type Client struct {
	phoneNumberID string
	accessToken   string
	http          *http.Client
}

// New returns a ready-to-use Client.
func New(phoneNumberID, accessToken string) *Client {
	return &Client{
		phoneNumberID: phoneNumberID,
		accessToken:   accessToken,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

// ─────────────────────────────────────────────
// Payload types
// ─────────────────────────────────────────────

type textBody struct {
	MessagingProduct string   `json:"messaging_product"`
	To               string   `json:"to"`
	Type             string   `json:"type"`
	Text             textObj  `json:"text"`
}

type textObj struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

// Interactive message (buttons / list)
type interactiveBody struct {
	MessagingProduct string        `json:"messaging_product"`
	To               string        `json:"to"`
	Type             string        `json:"type"`
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
	Type string `json:"type"`
	Text string `json:"text"`
}

type interactText struct {
	Text string `json:"text"`
}

type interactAction struct {
	// For button type
	Buttons []interactButton `json:"buttons,omitempty"`
	// For list type
	ButtonText string         `json:"button,omitempty"`
	Sections   []listSection  `json:"sections,omitempty"`
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
	Title string     `json:"title"`
	Rows  []listRow  `json:"rows"`
}

type listRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ─────────────────────────────────────────────
// Send helpers
// ─────────────────────────────────────────────

// SendText sends a plain text message.
func (c *Client) SendText(to, text string) error {
	payload := textBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text:             textObj{Body: text},
	}
	return c.post(payload)
}

// SendButtons sends an interactive message with up to 3 quick-reply buttons.
// buttons is a slice of [id, title] pairs.
func (c *Client) SendButtons(to, bodyText string, buttons [][2]string) error {
	btns := make([]interactButton, 0, len(buttons))
	for _, b := range buttons {
		btns = append(btns, interactButton{
			Type:  "reply",
			Reply: buttonReply{ID: b[0], Title: truncate(b[1], 20)},
		})
	}
	payload := interactiveBody{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: interactiveObj{
			Type: "button",
			Body: interactText{Text: bodyText},
			Action: interactAction{
				Buttons: btns,
			},
		},
	}
	return c.post(payload)
}

// SendList sends an interactive list message (supports up to 10 rows per section).
// sections is a slice of (title, rows) where rows is [][2]string (id, title).
func (c *Client) SendList(to, bodyText, buttonLabel string, sections []ListSection) error {
	waSections := make([]listSection, 0, len(sections))
	for _, s := range sections {
		rows := make([]listRow, 0, len(s.Rows))
		for _, r := range s.Rows {
			rows = append(rows, listRow{
				ID:          r[0],
				Title:       truncate(r[1], 24),
				Description: r[2],
			})
		}
		waSections = append(waSections, listSection{
			Title: s.Title,
			Rows:  rows,
		})
	}
	payload := interactiveBody{
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
	}
	return c.post(payload)
}

// ListSection is the exported type used when calling SendList.
type ListSection struct {
	Title string
	Rows  [][3]string // [id, title, description]
}

// ─────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────

func (c *Client) post(payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	url := fmt.Sprintf("%s/%s/messages", apiBase, c.phoneNumberID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// truncate cuts a string to max rune length.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
