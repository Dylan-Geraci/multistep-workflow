package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/dylangeraci/flowforge/internal/metrics"
	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/dylangeraci/flowforge/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const streamName = "flowforge:steps"
const groupName = "workers"

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)).String()
}

type Pool struct {
	db                        *pgxpool.Pool
	rdb                       *redis.Client
	workerCount               int
	consumerName              string
	recoveryIntervalSecs      int
	recoveryIdleThresholdSecs int
	metrics                   *metrics.Metrics
	wg                        sync.WaitGroup
	cancel                    context.CancelFunc
}

func NewPool(db *pgxpool.Pool, rdb *redis.Client, workerCount, recoveryIntervalSecs, recoveryIdleThresholdSecs int, m *metrics.Metrics) *Pool {
	hostname, _ := os.Hostname()
	return &Pool{
		db:                        db,
		rdb:                       rdb,
		workerCount:               workerCount,
		consumerName:              fmt.Sprintf("worker-%s-%d", hostname, os.Getpid()),
		recoveryIntervalSecs:      recoveryIntervalSecs,
		recoveryIdleThresholdSecs: recoveryIdleThresholdSecs,
		metrics:                   m,
	}
}

func (p *Pool) Start(ctx context.Context) error {
	// Create consumer group (ignore BUSYGROUP error if it already exists)
	err := p.rdb.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, p.cancel = context.WithCancel(ctx)

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.consume(ctx, i)
	}

	// Launch crash recovery goroutine
	p.wg.Add(1)
	go p.recoverOrphaned(ctx)

	slog.Info("worker pool started",
		"workers", p.workerCount,
		"consumer", p.consumerName,
		"recovery_interval_secs", p.recoveryIntervalSecs,
	)
	return nil
}

func (p *Pool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	slog.Info("worker pool stopped")
}

func (p *Pool) consume(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := p.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: fmt.Sprintf("%s-%d", p.consumerName, id),
			Streams:  []string{streamName, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			slog.ErrorContext(ctx, "XREADGROUP error", "worker_id", id, "error", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				p.processMessage(ctx, msg)
			}
		}
	}
}

func (p *Pool) processMessage(ctx context.Context, msg redis.XMessage) {
	runID, _ := msg.Values["run_id"].(string)
	stepIndexStr, _ := msg.Values["step_index"].(string)
	attemptID, _ := msg.Values["attempt_id"].(string)
	attemptNumberStr, _ := msg.Values["attempt_number"].(string)

	var stepIndex int
	fmt.Sscanf(stepIndexStr, "%d", &stepIndex)

	attemptNumber := 1
	if attemptNumberStr != "" {
		fmt.Sscanf(attemptNumberStr, "%d", &attemptNumber)
	}

	// Extract trace context propagated through Redis
	carrier := propagation.MapCarrier{}
	if tp, ok := msg.Values["traceparent"].(string); ok {
		carrier.Set("traceparent", tp)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	tracer := otel.Tracer("flowforge")
	ctx, runSpan := tracer.Start(ctx, "workflow.run",
		trace.WithAttributes(
			attribute.String("run.id", runID),
			attribute.Int("step.index", stepIndex),
			attribute.Int("step.attempt", attemptNumber),
		),
	)
	defer runSpan.End()

	log := slog.With("run_id", runID, "step_index", stepIndex, "attempt", attemptNumber)

	// Fetch run status — skip if not pending/running
	var runStatus string
	err := p.db.QueryRow(ctx,
		`SELECT status FROM workflow_runs WHERE id = $1`, runID,
	).Scan(&runStatus)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch run status", "error", err)
		p.ack(ctx, msg.ID)
		return
	}
	if runStatus != "pending" && runStatus != "running" {
		p.ack(ctx, msg.ID)
		return
	}

	// Fetch step definition
	var workflowID, action string
	var stepConfig json.RawMessage
	err = p.db.QueryRow(ctx,
		`SELECT ws.action, ws.config, wr.workflow_id
		 FROM workflow_steps ws
		 JOIN workflow_runs wr ON wr.workflow_id = ws.workflow_id
		 WHERE wr.id = $1 AND ws.step_index = $2`,
		runID, stepIndex,
	).Scan(&action, &stepConfig, &workflowID)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch step definition", "error", err)
		p.markRunFailed(ctx, runID, fmt.Sprintf("step definition not found: %v", err))
		p.ack(ctx, msg.ID)
		return
	}

	log = log.With("action", action)

	// Update run → running
	_, err = p.db.Exec(ctx,
		`UPDATE workflow_runs SET status = 'running', current_step = $2, started_at = COALESCE(started_at, now())
		 WHERE id = $1`,
		runID, stepIndex,
	)
	if err != nil {
		log.ErrorContext(ctx, "failed to update run to running", "error", err)
	}
	p.publishEvent(ctx, runID, model.EventRunStatusChanged, map[string]interface{}{
		"status": "running", "step_index": stepIndex,
	})

	// Get current run context
	var runContext json.RawMessage
	p.db.QueryRow(ctx,
		`SELECT context FROM workflow_runs WHERE id = $1`, runID,
	).Scan(&runContext)
	if runContext == nil {
		runContext = json.RawMessage(`{}`)
	}

	// Execute action
	startedAt := time.Now().UTC()
	p.publishEvent(ctx, runID, model.EventStepStarted, map[string]interface{}{
		"step_index": stepIndex, "action": action, "attempt_number": attemptNumber,
	})

	stepCtx, stepSpan := tracer.Start(ctx, "step.execute",
		trace.WithAttributes(
			attribute.String("step.action", action),
			attribute.Int("step.index", stepIndex),
		),
	)
	output, execErr := ExecuteAction(stepCtx, action, stepConfig, runContext)
	stepSpan.End()

	completedAt := time.Now().UTC()
	durationMs := int(completedAt.Sub(startedAt).Milliseconds())

	stepExecID := newULID()
	status := "completed"
	var errMsg *string
	if execErr != nil {
		status = "failed"
		s := execErr.Error()
		errMsg = &s
	}
	if output == nil {
		output = json.RawMessage(`{}`)
	}

	// Record step metrics
	p.metrics.StepExecutionsTotal.WithLabelValues(action, status).Inc()
	p.metrics.StepExecutionDuration.WithLabelValues(action).Observe(float64(durationMs))

	// Insert step execution
	_, err = p.db.Exec(ctx,
		`INSERT INTO step_executions (id, run_id, step_index, attempt_id, attempt_number, action, status, input, output, error_message, duration_ms, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		stepExecID, runID, stepIndex, attemptID, attemptNumber, action, status, stepConfig, output, errMsg, durationMs, startedAt, completedAt,
	)
	if err != nil {
		log.ErrorContext(ctx, "failed to insert step execution", "error", err)
	}

	// ACK the message
	p.ack(ctx, msg.ID)

	if execErr != nil {
		log.WarnContext(ctx, "step failed", "error", execErr, "duration_ms", durationMs)
		p.publishEvent(ctx, runID, model.EventStepFailed, map[string]interface{}{
			"step_index": stepIndex, "attempt_number": attemptNumber, "error": execErr.Error(), "duration_ms": durationMs,
		})
		p.handleStepFailure(ctx, workflowID, runID, stepIndex, attemptNumber, execErr.Error())
		return
	}

	log.InfoContext(ctx, "step completed", "duration_ms", durationMs)
	p.publishEvent(ctx, runID, model.EventStepCompleted, map[string]interface{}{
		"step_index": stepIndex, "attempt_number": attemptNumber, "duration_ms": durationMs,
	})

	// Update run context with step output
	_, err = p.db.Exec(ctx,
		`UPDATE workflow_runs SET context = $2 WHERE id = $1`,
		runID, output,
	)
	if err != nil {
		log.ErrorContext(ctx, "failed to update run context", "error", err)
	}

	// Check if there are more steps
	var totalSteps int
	p.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_steps WHERE workflow_id = $1`, workflowID,
	).Scan(&totalSteps)

	nextStep := stepIndex + 1
	if nextStep < totalSteps {
		// XADD next step with trace context
		nextAttemptID := newULID()
		nextValues := map[string]interface{}{
			"run_id":         runID,
			"step_index":     fmt.Sprintf("%d", nextStep),
			"attempt_id":     nextAttemptID,
			"attempt_number": "1",
		}
		nextCarrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, nextCarrier)
		for _, key := range []string{"traceparent"} {
			if v := nextCarrier.Get(key); v != "" {
				nextValues[key] = v
			}
		}
		p.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: streamName,
			Values: nextValues,
		})
	} else {
		// Last step — mark run completed
		now := time.Now().UTC()
		_, err = p.db.Exec(ctx,
			`UPDATE workflow_runs SET status = 'completed', completed_at = $2 WHERE id = $1`,
			runID, now,
		)
		if err != nil {
			log.ErrorContext(ctx, "failed to mark run completed", "error", err)
		}
		p.metrics.WorkflowRunsTotal.WithLabelValues("completed").Inc()
		slog.InfoContext(ctx, "run completed", "run_id", runID)
		p.publishEvent(ctx, runID, model.EventRunCompleted, map[string]interface{}{
			"status": "completed",
		})
	}
}

func (p *Pool) handleStepFailure(ctx context.Context, workflowID, runID string, stepIndex, attemptNumber int, errMsg string) {
	// Fetch retry policy from workflow
	var retryPolicyRaw json.RawMessage
	err := p.db.QueryRow(ctx,
		`SELECT retry_policy FROM workflows WHERE id = $1`, workflowID,
	).Scan(&retryPolicyRaw)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch retry policy", "workflow_id", workflowID, "error", err)
		p.markRunFailed(ctx, runID, errMsg)
		return
	}

	var policy model.RetryPolicy
	if retryPolicyRaw != nil {
		if err := json.Unmarshal(retryPolicyRaw, &policy); err != nil {
			slog.ErrorContext(ctx, "failed to parse retry policy", "error", err)
			p.markRunFailed(ctx, runID, errMsg)
			return
		}
	}
	policy = policy.WithDefaults()

	// attempt 1 = initial try, retries are attempts 2..maxRetries+1
	if attemptNumber >= policy.MaxRetries+1 {
		p.markRunFailed(ctx, runID, fmt.Sprintf("step %d failed after %d attempts: %s", stepIndex, attemptNumber, errMsg))
		return
	}

	// Compute exponential backoff delay
	delayMs := CalculateBackoffDelay(policy, attemptNumber)

	slog.InfoContext(ctx, "retrying step",
		"run_id", runID,
		"step_index", stepIndex,
		"attempt", attemptNumber,
		"next_attempt", attemptNumber+1,
		"delay_ms", delayMs,
	)

	// Context-aware sleep
	select {
	case <-time.After(time.Duration(delayMs) * time.Millisecond):
	case <-ctx.Done():
		return
	}

	// Re-check run status after backoff — may have been cancelled
	var currentStatus string
	if err := p.db.QueryRow(ctx, `SELECT status FROM workflow_runs WHERE id = $1`, runID).Scan(&currentStatus); err != nil {
		slog.ErrorContext(ctx, "failed to re-check run status", "run_id", runID, "error", err)
		return
	}
	if currentStatus != "pending" && currentStatus != "running" {
		slog.InfoContext(ctx, "run status changed during backoff, skipping retry",
			"run_id", runID, "status", currentStatus)
		return
	}

	// XADD retry with incremented attempt number and trace context
	nextAttemptID := newULID()
	retryValues := map[string]interface{}{
		"run_id":         runID,
		"step_index":     fmt.Sprintf("%d", stepIndex),
		"attempt_id":     nextAttemptID,
		"attempt_number": fmt.Sprintf("%d", attemptNumber+1),
	}
	retryCarrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, retryCarrier)
	if v := retryCarrier.Get("traceparent"); v != "" {
		retryValues["traceparent"] = v
	}
	p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: retryValues,
	})
}

func (p *Pool) recoverOrphaned(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Duration(p.recoveryIntervalSecs) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.doRecovery(ctx)
		}
	}
}

func (p *Pool) doRecovery(ctx context.Context) {
	pending, err := p.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: streamName,
		Group:  groupName,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()
	if err != nil {
		slog.ErrorContext(ctx, "XPENDING error during recovery", "error", err)
		return
	}

	idleThreshold := time.Duration(p.recoveryIdleThresholdSecs) * time.Second

	for _, msg := range pending {
		if msg.Idle < idleThreshold {
			continue
		}

		slog.InfoContext(ctx, "claiming orphaned message", "msg_id", msg.ID, "idle", msg.Idle)

		claimed, err := p.rdb.XClaim(ctx, &redis.XClaimArgs{
			Stream:   streamName,
			Group:    groupName,
			Consumer: p.consumerName,
			MinIdle:  idleThreshold,
			Messages: []string{msg.ID},
		}).Result()
		if err != nil {
			slog.ErrorContext(ctx, "XCLAIM error", "msg_id", msg.ID, "error", err)
			continue
		}

		for _, claimedMsg := range claimed {
			attemptID, _ := claimedMsg.Values["attempt_id"].(string)

			// Idempotency check: see if this attempt was already processed
			var exists bool
			err := p.db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM step_executions WHERE attempt_id = $1)`, attemptID,
			).Scan(&exists)
			if err != nil {
				slog.ErrorContext(ctx, "idempotency check error", "attempt_id", attemptID, "error", err)
				continue
			}

			if exists {
				// Already processed — just ACK
				p.ack(ctx, claimedMsg.ID)
				slog.InfoContext(ctx, "message already processed, ACKed", "msg_id", claimedMsg.ID, "attempt_id", attemptID)
			} else {
				slog.InfoContext(ctx, "reprocessing message", "msg_id", claimedMsg.ID)
				p.processMessage(ctx, claimedMsg)
			}
		}
	}
}

func (p *Pool) markRunFailed(ctx context.Context, runID, errMsg string) {
	p.metrics.WorkflowRunsTotal.WithLabelValues("failed").Inc()
	now := time.Now().UTC()
	_, err := p.db.Exec(ctx,
		`UPDATE workflow_runs SET status = 'failed', error_message = $2, completed_at = $3 WHERE id = $1`,
		runID, errMsg, now,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to mark run as failed", "run_id", runID, "error", err)
	}
	p.publishEvent(ctx, runID, model.EventRunFailed, map[string]interface{}{
		"status": "failed", "error": errMsg,
	})
}

func (p *Pool) ack(ctx context.Context, msgID string) {
	p.rdb.XAck(ctx, streamName, groupName, msgID)
}

func (p *Pool) publishEvent(ctx context.Context, runID string, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal event data", "error", err)
		return
	}
	event := model.WSEvent{
		Type:  eventType,
		RunID: runID,
		Data:  jsonData,
	}
	if err := ws.PublishEvent(ctx, p.rdb, runID, event); err != nil {
		slog.ErrorContext(ctx, "failed to publish event", "run_id", runID, "error", err)
	}
}
