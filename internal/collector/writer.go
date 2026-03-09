package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

const (
	batchSize     = 1000
	flushInterval = 5 * time.Second
	maxRetries    = 3
	retryDelay    = 2 * time.Second
)

type Writer struct {
	conn   driver.Conn
	log    *slog.Logger
	mu     sync.Mutex
	buffer []ClickEvent
	stopCh chan struct{}
	doneCh chan struct{}
}

func NewWriter(conn driver.Conn, log *slog.Logger) *Writer {
	w := &Writer{
		conn:   conn,
		log:    log,
		buffer: make([]ClickEvent, 0, batchSize),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go w.flushLoop()
	return w
}

func (w *Writer) Add(event ClickEvent) {
	w.mu.Lock()
	w.buffer = append(w.buffer, event)
	shouldFlush := len(w.buffer) >= batchSize
	w.mu.Unlock()

	if shouldFlush {
		w.flush()
	}
}

func (w *Writer) Stop() {
	close(w.stopCh)
	<-w.doneCh // wait for flushLoop to finish its current flush
	w.flush()  // final drain
}

func (w *Writer) flushLoop() {
	defer close(w.doneCh)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.flush()
		case <-w.stopCh:
			return
		}
	}
}

func (w *Writer) flush() {
	w.mu.Lock()
	if len(w.buffer) == 0 {
		w.mu.Unlock()
		return
	}
	events := w.buffer
	w.buffer = make([]ClickEvent, 0, batchSize)
	w.mu.Unlock()

	for attempt := range maxRetries {
		if err := w.sendBatch(events); err != nil {
			w.log.Error("failed to send batch",
				"error", err,
				"attempt", attempt+1,
				"events", len(events),
			)
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay * time.Duration(attempt+1))
				continue
			}
			// All retries exhausted — put events back to buffer
			w.mu.Lock()
			w.buffer = append(events, w.buffer...)
			w.mu.Unlock()
			w.log.Error("batch returned to buffer after all retries failed", "events", len(events))
			return
		}

		w.log.Info("flushed click events", "count", len(events))
		return
	}
}

func (w *Writer) sendBatch(events []ClickEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	batch, err := w.conn.PrepareBatch(ctx,
		"INSERT INTO snip.click_events (event_id, code, ip, user_agent, referer, timestamp)",
	)
	if err != nil {
		return err
	}

	for _, e := range events {
		ts := time.Unix(e.Timestamp, 0)
		if err := batch.Append(uuid.New(), e.Code, e.IP, e.UserAgent, e.Referer, ts); err != nil {
			return err
		}
	}

	return batch.Send()
}
