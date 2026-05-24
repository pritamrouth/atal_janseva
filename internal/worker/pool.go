// Package worker provides an async message-processing pool.
//
// The webhook handler enqueues inbound messages into a buffered channel and
// returns HTTP 200 immediately (as Meta requires). A fixed pool of worker
// goroutines drains the channel and calls the bot handler.
//
// This completely decouples webhook latency from DB / WhatsApp API latency,
// allowing the server to sustain thousands of concurrent conversations.
package worker

import (
	"log/slog"

	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
)

// Job is a single unit of work: one inbound WhatsApp message.
type Job struct {
	Msg whatsapp.Message
}

// Handler is the function signature the bot must implement.
type Handler func(msg whatsapp.Message)

// Pool manages a fixed set of goroutines that drain a shared job queue.
type Pool struct {
	queue   chan Job
	handler Handler
	workers int
}

// New creates a Pool with `workers` goroutines and a buffered queue of
// `queueDepth` jobs. Call Start() to launch the goroutines.
func New(workers, queueDepth int, handler Handler) *Pool {
	return &Pool{
		queue:   make(chan Job, queueDepth),
		handler: handler,
		workers: workers,
	}
}

// Start launches the worker goroutines. It is non-blocking.
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		go p.run(i)
	}
	slog.Info("worker pool started", "workers", p.workers, "queue_depth", cap(p.queue))
}

// Enqueue submits a job. If the queue is full it drops the message and logs a
// warning rather than blocking the HTTP handler goroutine.
func (p *Pool) Enqueue(msg whatsapp.Message) {
	select {
	case p.queue <- Job{Msg: msg}:
	default:
		slog.Warn("worker queue full – message dropped",
			"from", msg.From, "type", msg.Type)
	}
}

// QueueLen returns the current number of pending jobs (useful for metrics).
func (p *Pool) QueueLen() int { return len(p.queue) }

func (p *Pool) run(id int) {
	slog.Debug("worker started", "id", id)
	for job := range p.queue {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("worker panic recovered",
						"worker_id", id,
						"from", job.Msg.From,
						"panic", r,
					)
				}
			}()
			p.handler(job.Msg)
		}()
	}
}
