package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
)

type LeonardoWebhookProcessor struct {
	events        LeonardoWebhookEventRepository
	orchestrator  *LeonardoGenerationPollOrchestrator
	interval      time.Duration
	batchSize     int
	mu            sync.Mutex
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	ops           OpsRepository
	lastClaimed   int
	lastProcessed int
	lastFailed    int
}

func NewLeonardoWebhookProcessor(events LeonardoWebhookEventRepository, orchestrator *LeonardoGenerationPollOrchestrator, ops ...OpsRepository) *LeonardoWebhookProcessor {
	processor := &LeonardoWebhookProcessor{events: events, orchestrator: orchestrator, interval: time.Second, batchSize: 50}
	if len(ops) > 0 {
		processor.ops = ops[0]
	}
	return processor
}

func (p *LeonardoWebhookProcessor) Start() {
	if p == nil || p.events == nil || p.orchestrator == nil {
		return
	}
	p.mu.Lock()
	if p.cancel != nil {
		p.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.wg.Add(1)
	p.mu.Unlock()
	go p.run(ctx)
}

func (p *LeonardoWebhookProcessor) Stop() {
	if p == nil {
		return
	}
	p.mu.Lock()
	cancel := p.cancel
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.wg.Wait()
}

func (p *LeonardoWebhookProcessor) run(ctx context.Context) {
	defer p.wg.Done()
	p.runOnce(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.runOnce(ctx)
		}
	}
}

func (p *LeonardoWebhookProcessor) runOnce(ctx context.Context) {
	started := time.Now().UTC()
	err := p.RunOnce(ctx)
	p.heartbeat(started, err)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("[LeonardoWebhook] Batch failed: %v", err)
	}
}

func (p *LeonardoWebhookProcessor) RunOnce(ctx context.Context) error {
	p.lastClaimed, p.lastProcessed, p.lastFailed = 0, 0, 0
	events, err := p.events.ClaimPending(ctx, p.batchSize)
	if err != nil {
		return err
	}
	p.lastClaimed = len(events)
	for _, event := range events {
		if event == nil {
			continue
		}
		if err = p.process(ctx, event); err != nil {
			p.lastFailed++
			_ = p.events.MarkFailed(context.WithoutCancel(ctx), event.ID, event.AttemptCount+1)
			continue
		}
		if err = p.events.MarkProcessed(ctx, event.ID); err != nil {
			return err
		}
		p.lastProcessed++
	}
	return nil
}

func (p *LeonardoWebhookProcessor) heartbeat(started time.Time, runErr error) {
	if p.ops == nil {
		return
	}
	now, duration := time.Now().UTC(), time.Since(started).Milliseconds()
	input := &OpsUpsertJobHeartbeatInput{JobName: "leonardo_webhook_worker", LastRunAt: &now, LastDurationMs: &duration}
	if runErr != nil {
		message := runErr.Error()
		input.LastErrorAt, input.LastError = &now, &message
	} else {
		input.LastSuccessAt = &now
		summary := fmt.Sprintf("claimed=%d processed=%d failed=%d", p.lastClaimed, p.lastProcessed, p.lastFailed)
		input.LastResult = &summary
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = p.ops.UpsertJobHeartbeat(ctx, input)
}

func (p *LeonardoWebhookProcessor) process(ctx context.Context, event *LeonardoWebhookEvent) error {
	if event.UpstreamGenerationID == "" {
		return ErrLeonardoWebhookInvalid
	}
	var payload struct {
		Data struct {
			Object struct {
				ID     string                    `json:"id"`
				Status string                    `json:"status"`
				Images []leonardo.GeneratedImage `json:"images"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	generation := &leonardo.Generation{ID: payload.Data.Object.ID, Status: payload.Data.Object.Status, GeneratedImages: payload.Data.Object.Images}
	if generation.ID == "" {
		generation.ID = event.UpstreamGenerationID
	}
	_, err := p.orchestrator.ApplyWebhook(ctx, event.AccountID, generation)
	return err
}
