package services

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func evaluateTransferCheck(ctx context.Context, tx pgx.Tx, data kafka.FraudCheckRequestedData) error {
	transfer, err := EvaluateTransferRules(ctx, tx, data)
	if err != nil {
		return err
	}
	graph, err := EvaluateGraphRules(ctx, tx, data)
	if err != nil {
		return err
	}
	geo, err := repositories.EvaluateGeoRule(ctx, tx, data)
	if err != nil {
		return err
	}

	rules := append(transfer, graph...)
	rules = append(rules, geo)
	return repositories.SaveCheck(ctx, tx, data.TransferID, rules)
}

func recoverUncheckedTransfers(ctx context.Context, pool *pgxpool.Pool) error {
	for ctx.Err() == nil {
		requests, err := repositories.ListUncheckedTransferRequests(ctx, pool)
		if err != nil {
			return fmt.Errorf("read transfer requests without a check: %w", err)
		}
		if len(requests) == 0 {
			return nil
		}
		for _, request := range requests {
			event, err := kafka.Decode[kafka.FraudCheckRequestedData](request.Payload, kafka.EventFraudCheckRequested)
			if err != nil {
				return err
			}
			if event.Data.TransferID != request.TransferID {
				return fmt.Errorf("inbox event %s does not match its transfer %d", event.EventID, request.TransferID)
			}
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err = repositories.StoreTransferAttempt(checkCtx, pool, event, request.Payload,
				models.Geo{Status: models.GeoMissing}, evaluateTransferCheck)
			cancel()
			if err != nil {
				return err
			}
		}
	}
	return ctx.Err()
}
