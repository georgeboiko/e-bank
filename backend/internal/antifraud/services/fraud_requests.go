package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/twmb/franz-go/pkg/kgo"
)

type GeoResolver interface {
	Resolve(context.Context, netip.Addr) models.Geo
}

func RunInboxConsumer(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, client *kgo.Client, resolver GeoResolver) error {
	if err := recoverUncheckedTransfers(ctx, pool); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("recover unfinished Antifraud checks: %w", err)
	}

	for ctx.Err() == nil {
		fetches := client.PollRecords(ctx, 1)
		records := fetches.RecordIter()
		if records.Done() {
			client.AllowRebalance()
			if err := fetches.Err(); err != nil && ctx.Err() == nil {
				logger.Warn("Antifraud Kafka poll failed", "error", err)
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
				}
			}
			continue
		}

		record := records.Next()
		err := applyInboxRecord(ctx, pool, client, resolver, record)
		client.AllowRebalance()
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Antifraud Kafka partition %d offset %d: %w", record.Partition, record.Offset, err)
		}
	}
	return nil
}

func applyInboxRecord(ctx context.Context, pool *pgxpool.Pool, client *kgo.Client, resolver GeoResolver, record *kgo.Record) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var err error
	switch record.Topic {
	case kafka.TopicFraudCheckRequested:
		err = applyFraudRequest(ctx, pool, resolver, record)
	case kafka.TopicLoginSucceeded:
		err = applyLoginSucceeded(ctx, pool, resolver, record)
	default:
		return fmt.Errorf("unexpected inbox topic %q", record.Topic)
	}
	if err != nil {
		return err
	}
	if err := client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit Kafka offset: %w", err)
	}
	return nil
}

func applyFraudRequest(ctx context.Context, pool *pgxpool.Pool, resolver GeoResolver, record *kgo.Record) error {
	event, err := kafka.Decode[kafka.FraudCheckRequestedData](record.Value, kafka.EventFraudCheckRequested)
	if err != nil {
		return err
	}
	if err := validateFraudRequest(event); err != nil {
		return err
	}
	geo := models.Geo{Status: models.GeoMissing}
	if event.Data.IPAddress != nil {
		geo = resolver.Resolve(ctx, *event.Data.IPAddress)
	}
	return repositories.StoreTransferAttempt(ctx, pool, event, record.Value, geo, evaluateTransferCheck)
}

func validateFraudRequest(event kafka.FraudCheckRequested) error {
	data := event.Data
	if event.OccurredAt.IsZero() || data.CreatedAt.IsZero() {
		return fmt.Errorf("FraudCheckRequested requires occurredAt and createdAt")
	}
	if data.TransferID < 1 || data.FromAccountID < 1 || data.ToAccountID < 1 || data.FromAccountID == data.ToAccountID {
		return fmt.Errorf("FraudCheckRequested has invalid transfer or account IDs")
	}
	if strings.TrimSpace(data.SenderID) == "" || strings.TrimSpace(data.RecipientID) == "" {
		return fmt.Errorf("FraudCheckRequested requires senderId and recipientId")
	}
	if !validTransferAmount(data.Amount) {
		return fmt.Errorf("FraudCheckRequested amount must be positive and fit numeric(19,2)")
	}
	if data.DeviceID != nil && strings.TrimSpace(*data.DeviceID) == "" {
		return fmt.Errorf("FraudCheckRequested deviceId cannot be blank")
	}
	if data.IPAddress != nil && (!data.IPAddress.IsValid() || data.IPAddress.Zone() != "") {
		return fmt.Errorf("FraudCheckRequested ipAddress must be an IPv4 or IPv6 address")
	}
	return nil
}

func validTransferAmount(amount decimal.Decimal) bool {
	if !amount.IsPositive() {
		return false
	}
	digits := amount.Coefficient().String()
	scale := int64(amount.Exponent())
	trailingZeros := int64(len(digits) - len(strings.TrimRight(digits, "0")))
	return int64(len(digits))+scale <= 17 && scale+trailingZeros >= -2
}
