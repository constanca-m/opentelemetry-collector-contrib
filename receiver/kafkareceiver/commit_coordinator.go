// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// commitCoordinator stores the latest uncommitted record per partition.
// Workers report offsets and continue. An entry is valid only while this
// consumer owns that partition. Assignment exit paths must flush the offset
// (revocation) or drop it (fatal loss).
type commitCoordinator struct {
	mu      sync.Mutex
	pending map[topicPartition]*kgo.Record
	notify  chan struct{}

	// ctx is the parent of every CommitRecords attempt. Shutdown cancels it
	// so a later resume cannot commit against a closed client.
	ctx context.Context
	// cancel ends ctx. Call it once from Shutdown.
	cancel context.CancelFunc
	// inFlightCancel ends the current attempt. pause uses it. nil means no
	// attempt is running.
	inFlightCancel context.CancelFunc
	// paused blocks startAttempt. lost() sets it for the cleanup window and
	// then clears it. It is a boolean, not a counter, because franz-go runs
	// one group callback at a time.
	paused bool
	// wg tracks the commit loop goroutine. wait() blocks until it exits.
	wg sync.WaitGroup
}

func newCommitCoordinator(ctx context.Context) *commitCoordinator {
	ctx, cancel := context.WithCancel(ctx)
	return &commitCoordinator{
		pending: make(map[topicPartition]*kgo.Record),
		notify:  make(chan struct{}, 1),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// report stores rec when it is newer than the pending record for tp.
// It then wakes the commit loop.
func (c *commitCoordinator) report(tp topicPartition, rec *kgo.Record) {
	c.mu.Lock()
	c.pending[tp] = newerRecord(c.pending[tp], rec)
	c.mu.Unlock()
	c.kick()
}

// kick wakes the commit loop. A second kick is a no-op while the loop
// has not yet consumed the first.
func (c *commitCoordinator) kick() {
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

// takeAll removes every pending record and returns them.
func (c *commitCoordinator) takeAll() []*kgo.Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	records := make([]*kgo.Record, 0, len(c.pending))
	for _, rec := range c.pending {
		records = append(records, rec)
	}
	clear(c.pending)
	return records
}

// take removes and returns the pending record for each named partition.
// An empty or nil list returns no records and leaves pending records unchanged.
func (c *commitCoordinator) take(tps []topicPartition) []*kgo.Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	records := make([]*kgo.Record, 0, len(tps))
	for _, tp := range tps {
		rec, ok := c.pending[tp]
		if !ok {
			continue
		}
		records = append(records, rec)
		delete(c.pending, tp)
	}
	return records
}

// restore puts records back after a failed commit. It keeps a newer
// report that arrived while the commit ran. It does not wake the loop.
// The next report or a revocation flush retries.
func (c *commitCoordinator) restore(records []*kgo.Record) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, rec := range records {
		tp := topicPartition{topic: rec.Topic, partition: rec.Partition}
		c.pending[tp] = newerRecord(c.pending[tp], rec)
	}
}

// stop cancels the shutdown context so an in-flight CommitRecords cannot
// continue against a closed client.
func (c *commitCoordinator) stop() {
	c.cancel()
}

// startAttempt creates a bounded context for one CommitRecords call.
// It returns ok=false when the loop is paused.
func (c *commitCoordinator) startAttempt(timeout time.Duration) (context.Context, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.paused {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	c.inFlightCancel = cancel
	return ctx, true
}

// endAttempt cancels and clears the current attempt.
func (c *commitCoordinator) endAttempt() {
	c.mu.Lock()
	cancel := c.inFlightCancel
	c.inFlightCancel = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// withAttempt runs fn with a bounded commit context. It returns ok=false when
// the loop is paused. Callers must hold opsMu.
func (c *commitCoordinator) withAttempt(timeout time.Duration, fn func(context.Context) error) (bool, error) {
	ctx, ok := c.startAttempt(timeout)
	if !ok {
		return false, nil
	}
	defer c.endAttempt()
	return true, fn(ctx)
}

// pause blocks new attempts and cancels the current one. lost() calls pause
// around cleanup. paused is set before endAttempt so startAttempt cannot
// start a new commit in the gap.
func (c *commitCoordinator) pause() {
	c.mu.Lock()
	c.paused = true
	c.mu.Unlock()
	c.endAttempt()
}

// resume allows new attempts and wakes the loop.
func (c *commitCoordinator) resume() {
	c.mu.Lock()
	c.paused = false
	c.mu.Unlock()
	c.kick()
}

// start runs the commit loop until closing is closed.
func (c *commitCoordinator) start(closing <-chan struct{}, commit func()) {
	c.wg.Go(func() {
		for {
			select {
			case <-closing:
				return
			case <-c.notify:
				commit()
			}
		}
	})
}

func (c *commitCoordinator) wait() {
	c.wg.Wait()
}

// newerRecord returns the record with the greater offset. A nil current
// yields incoming.
func newerRecord(current, incoming *kgo.Record) *kgo.Record {
	if current == nil || incoming.Offset >= current.Offset {
		return incoming
	}
	return current
}
