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
	configs, err := repositories.ReadRuleConfigs(ctx, tx)
	if err != nil {
		return err
	}
	transfer, err := EvaluateTransferRules(ctx, tx, data, configs)
	if err != nil {
		return err
	}
	graph, err := EvaluateGraphRules(ctx, tx, data, configs)
	if err != nil {
		return err
	}
	geo := disabledRule(8)
	if configs[7].Enabled {
		geo, err = repositories.EvaluateGeoRule(ctx, tx, data, configs[7])
		if err != nil {
			return err
		}
	}

	rules := append(transfer, graph...)
	rules = append(rules, geo)
	for i, rule := range rules {
		if !configs[rule.RuleID-1].Enabled {
			rules[i] = disabledRule(rule.RuleID)
		}
	}
	return repositories.SaveCheck(ctx, tx, data.TransferID, rules)
}

func disabledRule(id int32) models.RuleEvaluation {
	return models.RuleEvaluation{RuleID: id, Status: models.RuleSkipped, Reason: "Правило выключено"}
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
