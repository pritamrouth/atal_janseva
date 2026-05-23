# Ataljanseva WhatsApp Cloud API Bot — Go Backend

A production-ready Go backend that implements the **Ataljanseva Citizen Service** onboarding flow
over the **Meta WhatsApp Cloud API**.

## Flow overview

```
User sends any message
        │
        ▼
[ Step 0 ] Language picker        → buttons: English / मराठी / हिंदी
        │ tap language button
        ▼
[ Step 1 ] Enter PIN code         → free-text: 411001 / 400001 / 440001 / 421301
        │ valid PIN
        ▼
[ Step 2a ] Ward selection        → list message (state + district shown)
        │ pick ward
        ▼
[ Step 2b ] Nagarsevak selection  → list message (name + party shown)
        │ pick nagarsevak
        ▼
[ Step 3 ] Main menu              → buttons: SOS / Register complaint / Track
        │ tap action
        ▼
[ Sub-flow ] SOS / Register / Track  ← plug your own handlers here
```

At any point, typing **`reset`** clears the session and restarts the flow.

---

## Project structure

```
.
├── cmd/server/
│   ├── main.go          – wires everything; starts HTTP server
│   └── webhook.go       – GET (hub verify) + POST (inbound messages)
├── config/
│   └── config.go        – reads env vars
├── internal/
│   ├── bot/
│   │   ├── data.go      – static pincode / nagarsevak data
│   │   ├── handler.go   – state-machine: drives the conversation
│   │   └── i18n.go      – EN / MR / HI strings
│   ├── store/
│   │   └── store.go     – in-memory session store (swap for Redis in prod)
│   └── whatsapp/
│       ├── client.go    – sends text / button / list messages
│       └── payload.go   – unmarshals inbound webhook payloads
├── .env.example
├── Dockerfile
└── go.mod
```

---

## Prerequisites

| Tool | Version |
|------|---------|
| Go   | ≥ 1.22  |
| Meta Developer account | — |
| WhatsApp Business Account | — |
| Public HTTPS URL for the webhook | (ngrok for local dev) |

---

## Local development

### 1. Clone & install deps

```bash
git clone https://github.com/your-org/ataljanseva-wa-bot
cd ataljanseva-wa-bot
go mod tidy
```

### 2. Configure environment

```bash
cp .env.example .env
# Edit .env with your real values:
#   WA_PHONE_NUMBER_ID  – from Meta dashboard → WhatsApp → API Setup
#   WA_ACCESS_TOKEN     – system-user permanent token
#   WA_VERIFY_TOKEN     – any secret string you choose
```

### 3. Run the server

```bash
go run ./cmd/server
# → [main] Ataljanseva WhatsApp Bot listening on :8080
```

### 4. Expose locally with ngrok

```bash
ngrok http 8080
# Copy the https://xxxx.ngrok.io URL
```

### 5. Register the webhook in Meta Dashboard

1. Go to **Meta Developers → your app → WhatsApp → Configuration**.
2. Set **Callback URL** to `https://xxxx.ngrok.io/webhook`.
3. Set **Verify Token** to the value of `WA_VERIFY_TOKEN` in your `.env`.
4. Click **Verify and Save**.
5. Subscribe to the **messages** field.

---

## Production deployment (Docker)

```bash
# Build
docker build -t ataljanseva-wa-bot:latest .

# Run (pass env vars via --env-file or -e)
docker run -d \
  --env-file .env \
  -p 8080:8080 \
  --name ataljanseva-wa-bot \
  ataljanseva-wa-bot:latest
```

Deploy behind **nginx** or a cloud load-balancer that terminates TLS — Meta requires HTTPS for webhooks.

---

## Extending sub-flows

Open `internal/bot/handler.go` and find `handleMainMenuSelection`. Replace the placeholder `SendText`
confirmation with your own sub-flow handler:

```go
case "action_sos":
    // e.g. start SOS complaint flow
    return h.startSOSFlow(phone, sess)
case "action_register":
    return h.startRegisterFlow(phone, sess)
case "action_track":
    return h.startTrackFlow(phone, sess)
```

Each sub-flow can add new `Step` constants in `internal/store/store.go` and new `Pending` keys to
route list/button replies.

---

## Session persistence

`internal/store/store.go` uses an in-memory `sync.Map`. For multi-instance or persistent sessions,
replace the `Store` struct with a Redis or database backend — the interface is just `Get`, `Save`,
and `Reset`.

---

## Health check

```
GET /health
→ {"status":"ok","service":"ataljanseva-wa-bot"}
```

---

## Demo PIN codes

| PIN    | Location     |
|--------|-------------|
| 411001 | Pune         |
| 400001 | Mumbai City  |
| 440001 | Nagpur       |
| 421301 | Thane        |
