package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/augno/api/services/agent-service/internal/domain"
	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/lease"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
	"github.com/robfig/cron/v3"
)

var schedulerTracer = tracing.GetTracer("agent-service.scheduler")

const (
	schedulerLeaseName = "agent-scheduler"
	schedulerLeaseTTL  = 90 * time.Second
)

type SchedulerConfig struct {
	// Repos (required) is the repository factory for agent persistence.
	Repos domain.RepoFactory

	// OutboxRepo (required) is the outbox repository used to enqueue run commands.
	OutboxRepo messaging.OutboxRepo

	// PollInterval (optional; default: 60s) controls how frequently the scheduler polls for due agent schedules.
	PollInterval time.Duration

	// PlanGate (optional; default: nil) checks whether an account's plan allows agents. When nil, plan gating is skipped and all accounts are allowed.
	PlanGate PlanGate

	// Lease (required) is the distributed lease ensuring only one pod schedules runs per tick.
	Lease *lease.Lease
}

// WithDefaults fills zero-value fields with production defaults and returns the config.
func (c *SchedulerConfig) WithDefaults() *SchedulerConfig {
	if c == nil {
		c = &SchedulerConfig{}
	}

	if c.PollInterval == 0 {
		c.PollInterval = 60 * time.Second
	}
	return c
}

func (c *SchedulerConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("scheduler service: repos is required")
	}
	if c.OutboxRepo == nil {
		return fmt.Errorf("scheduler service: outbox repo is required")
	}
	if c.Lease == nil {
		return fmt.Errorf("scheduler service: lease is required")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("scheduler service: poll interval must be positive")
	}
	return nil
}

type schedulerSvc struct {
	repos        domain.RepoFactory
	outboxRepo   messaging.OutboxRepo
	pollInterval time.Duration
	planGate     PlanGate
	lease        *lease.Lease
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewSchedulerSvc(config *SchedulerConfig) domain.SchedulerSvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &schedulerSvc{
		repos:        config.Repos,
		outboxRepo:   config.OutboxRepo,
		pollInterval: config.PollInterval,
		planGate:     config.PlanGate,
		lease:        config.Lease,
		stopCh:       make(chan struct{}),
	}
}

func (s *schedulerSvc) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.pollLoop(ctx)
	slog.Info("Agent scheduler started", "poll_interval", s.pollInterval)
	return nil
}

func (s *schedulerSvc) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Agent scheduler stopped")
}

func (s *schedulerSvc) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			_ = s.lease.WithLease(ctx, schedulerLeaseName, schedulerLeaseTTL, func(leaseCtx context.Context) error {
				s.checkSchedules(leaseCtx)
				return nil
			})
		}
	}
}

func (s *schedulerSvc) checkSchedules(ctx context.Context) {
	ctx, span := tracing.StartSpan(ctx, schedulerTracer, "service.scheduler.check_schedules")
	defer span.End()

	configRepo := s.repos.NewAgentConfigRepo()
	runRepo := s.repos.NewAgentRunRepo()

	configs, apiErr := configRepo.ListEnabledWithSchedule(ctx)
	if apiErr != nil {
		slog.Error("Scheduler: failed to list configs with schedule", "error", apiErr)
		return
	}

	now := time.Now()

	for _, cfg := range configs {
		if !cfg.Schedule.Valid || cfg.Schedule.String == "" {
			continue
		}

		schedule, parseErr := cron.ParseStandard(cfg.Schedule.String)
		if parseErr != nil {
			slog.Error("Scheduler: invalid cron expression",
				"config_id", cfg.ID,
				"schedule", cfg.Schedule.String,
				"error", parseErr,
			)
			continue
		}

		// Find last run for this config
		lastRun, lastRunAPIErr := runRepo.GetLastByConfigID(ctx, cfg.ID)
		var lastRunTime time.Time
		if lastRunAPIErr != nil {
			// No previous run — use config creation time as baseline
			lastRunTime = cfg.CreatedAt.Time
		} else {
			lastRunTime = lastRun.CreatedAt.Time
		}

		// Check if it's time for the next run
		nextRun := schedule.Next(lastRunTime)
		if !nextRun.Before(now) {
			continue
		}

		// Check plan eligibility before scheduling
		if s.planGate != nil {
			allowed, gateErr := s.planGate.CanUseAgents(ctx, cfg.AccountID)
			if gateErr != nil {
				slog.Warn("Scheduler: plan check failed, skipping run",
					"config_id", cfg.ID,
					"account_id", cfg.AccountID,
					"error", gateErr,
				)
				continue
			}
			if !allowed {
				slog.Warn("Scheduler: account not allowed to run agents, skipping",
					"config_id", cfg.ID,
					"account_id", cfg.AccountID,
				)
				continue
			}
		}

		// Schedule is due — create a new run
		if scheduleErr := s.scheduleRun(ctx, cfg); scheduleErr != nil {
			slog.Error("Scheduler: failed to schedule run",
				"config_id", cfg.ID,
				"error", scheduleErr,
			)
		}
	}
}

func (s *schedulerSvc) scheduleRun(ctx context.Context, cfg sqlc.ListEnabledConfigsWithScheduleRow) error {
	ctx, span := tracing.StartSpan(ctx, schedulerTracer, "service.scheduler.schedule_run")
	defer span.End()

	runRepo := s.repos.NewAgentRunRepo()

	runID, err := id.GenID(id.AgentRunIDPrefix, nil)
	if err != nil {
		return err
	}

	if insertErr := runRepo.Insert(ctx, sqlc.InsertAgentRunParams{
		ID:                runID,
		AccountID:         cfg.AccountID,
		AgentDefinitionID: cfg.AgentDefinitionID,
		AgentConfigID:     agentdb.PgText(cfg.ID),
		StatusCode:        domain.RunStatusPending,
		TriggerType:       string(constants.AgentTriggerTypeScheduled),
		Input:             json.RawMessage(`{}`),
		Output:            json.RawMessage(`{}`),
	}); insertErr != nil {
		return fmt.Errorf("failed to insert run: %w", insertErr)
	}

	// Write outbox message
	length := id.IDLength22
	msgID, err := id.GenID(id.MessageIDPrefix, &length)
	if err != nil {
		return err
	}

	data := messaging.AgentExecuteRunData{
		AgentRunID:    runID,
		AgentConfigID: cfg.ID,
		AccountID:     cfg.AccountID,
		TriggerType:   string(constants.AgentTriggerTypeScheduled),
	}
	dataBytes, _ := json.Marshal(data)

	if _, outboxErr := s.outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		MessageID:   msgID,
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.AgentCmdExecuteRun),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.AgentCmdExecuteRun),
		Payload: contracts.AmqpMessage{
			Data:      dataBytes,
			MessageID: msgID,
		},
		MaxAttempts: 3,
	}); outboxErr != nil {
		return outboxErr
	}

	slog.Info("Scheduler: created scheduled run",
		"run_id", runID,
		"config_id", cfg.ID,
		"agent_slug", cfg.DefinitionSlug,
	)

	return nil
}
