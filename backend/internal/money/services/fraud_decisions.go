package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/money/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func RunFraudDecisionConsumer(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, client *kgo.Client) error {
	for ctx.Err() == nil {
		fetches := client.PollRecords(ctx, 1)
		records := fetches.RecordIter()
		if records.Done() {
			client.AllowRebalance()
			if ctx.Err() != nil {
				return nil
			}
			if fetches.IsClientClosed() {
				return fmt.Errorf("Money Kafka consumer closed")
			}
			if err := fetches.Err(); err != nil {
				logger.Warn("Money Kafka poll failed", "error", err)
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
				}
			}
			continue
		}

		record := records.Next()
		err := applyFraudDecisionRecord(ctx, pool, client, record)
		client.AllowRebalance()
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Money Kafka partition %d offset %d: %w", record.Partition, record.Offset, err)
		}
	}
	return nil
}

func applyFraudDecisionRecord(ctx context.Context, pool *pgxpool.Pool, client *kgo.Client, record *kgo.Record) error {
	event, err := kafka.Decode[kafka.FraudDecisionData](record.Value, kafka.EventFraudDecision)
	if err != nil {
		return err
	}
	if err := validateFraudDecision(event, string(record.Key)); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := repositories.ApplyFraudDecision(ctx, pool, record.Value); err != nil {
		return err
	}
	if err := client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit Kafka offset: %w", err)
	}
	return nil
}

func validateFraudDecision(event kafka.FraudDecision, key string) error {
	data := event.Data
	if event.OccurredAt.IsZero() || data.TransferID < 1 {
		return fmt.Errorf("FraudDecision requires occurredAt and a positive transferId")
	}
	if key != strconv.FormatInt(int64(data.TransferID), 10) {
		return fmt.Errorf("FraudDecision Kafka key does not match transferId")
	}
	if data.TriggeredRules == nil || len(data.TriggeredRules) > 8 {
		return fmt.Errorf("FraudDecision requires an array of at most eight triggered rules")
	}
	var seen [9]bool
	for _, rule := range data.TriggeredRules {
		if rule.RuleID < 1 || rule.RuleID > 8 || seen[rule.RuleID] ||
			rule.Score < 1 || rule.Score > 100 || strings.TrimSpace(rule.Reason) == "" {
			return fmt.Errorf("FraudDecision contains an invalid or repeated triggered rule")
		}
		seen[rule.RuleID] = true
	}
	return nil
}
