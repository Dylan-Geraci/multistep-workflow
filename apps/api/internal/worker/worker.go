package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
)

const streamName = "flowforge:steps"
const groupName = "workers"

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)).String()
}

type Pool struct {
	db           *pgxpool.Pool
	rdb          *redis.Client
	workerCount  int
	consumerName string
	wg           sync.WaitGroup
	cancel       context.CancelFunc
}

func NewPool(db *pgxpool.Pool, rdb *redis.Client, workerCount int) *Pool {
	hostname, _ := os.Hostname()
	return &Pool{
		db:           db,
		rdb:          rdb,
		workerCount:  workerCount,
		consumerName: fmt.Sprintf("worker-%s-%d", hostname, os.Getpid()),
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

	log.Printf("Worker pool started: %d workers, consumer=%s", p.workerCount, p.consumerName)
	return nil
}

func (p *Pool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	log.Println("Worker pool stopped")
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
			log.Printf("Worker %d: XREADGROUP error: %v", id, err)
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

	var stepIndex int
	fmt.Sscanf(stepIndexStr, "%d", &stepIndex)

	// Fetch run status — skip if not pending/running
	var runStatus string
	err := p.db.QueryRow(ctx,
		`SELECT status FROM workflow_runs WHERE id = $1`, runID,
	).Scan(&runStatus)
	if err != nil {
		log.Printf("Worker: failed to fetch run %s: %v", runID, err)
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
		log.Printf("Worker: failed to fetch step definition for run=%s step=%d: %v", runID, stepIndex, err)
		p.markRunFailed(ctx, runID, fmt.Sprintf("step definition not found: %v", err))
		p.ack(ctx, msg.ID)
		return
	}

	// Update run → running
	_, err = p.db.Exec(ctx,
		`UPDATE workflow_runs SET status = 'running', current_step = $2, started_at = COALESCE(started_at, now())
		 WHERE id = $1`,
		runID, stepIndex,
	)
	if err != nil {
		log.Printf("Worker: failed to update run %s to running: %v", runID, err)
	}

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
	output, execErr := ExecuteAction(ctx, action, stepConfig, runContext)
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

	// Insert step execution
	_, err = p.db.Exec(ctx,
		`INSERT INTO step_executions (id, run_id, step_index, attempt_id, attempt_number, action, status, input, output, error_message, duration_ms, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $9, $10, $11, $12)`,
		stepExecID, runID, stepIndex, attemptID, action, status, stepConfig, output, errMsg, durationMs, startedAt, completedAt,
	)
	if err != nil {
		log.Printf("Worker: failed to insert step execution: %v", err)
	}

	// ACK the message
	p.ack(ctx, msg.ID)

	if execErr != nil {
		p.markRunFailed(ctx, runID, execErr.Error())
		return
	}

	// Update run context with step output
	_, err = p.db.Exec(ctx,
		`UPDATE workflow_runs SET context = $2 WHERE id = $1`,
		runID, output,
	)
	if err != nil {
		log.Printf("Worker: failed to update run context: %v", err)
	}

	// Check if there are more steps
	var totalSteps int
	p.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_steps WHERE workflow_id = $1`, workflowID,
	).Scan(&totalSteps)

	nextStep := stepIndex + 1
	if nextStep < totalSteps {
		// XADD next step
		nextAttemptID := newULID()
		p.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: streamName,
			Values: map[string]interface{}{
				"run_id":     runID,
				"step_index": fmt.Sprintf("%d", nextStep),
				"attempt_id": nextAttemptID,
			},
		})
	} else {
		// Last step — mark run completed
		now := time.Now().UTC()
		_, err = p.db.Exec(ctx,
			`UPDATE workflow_runs SET status = 'completed', completed_at = $2 WHERE id = $1`,
			runID, now,
		)
		if err != nil {
			log.Printf("Worker: failed to mark run %s completed: %v", runID, err)
		}
	}
}

func (p *Pool) markRunFailed(ctx context.Context, runID, errMsg string) {
	now := time.Now().UTC()
	_, err := p.db.Exec(ctx,
		`UPDATE workflow_runs SET status = 'failed', error_message = $2, completed_at = $3 WHERE id = $1`,
		runID, errMsg, now,
	)
	if err != nil {
		log.Printf("Worker: failed to mark run %s as failed: %v", runID, err)
	}
}

func (p *Pool) ack(ctx context.Context, msgID string) {
	p.rdb.XAck(ctx, streamName, groupName, msgID)
}
