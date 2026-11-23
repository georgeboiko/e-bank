package services

import (
	"context"
	"fmt"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func EvaluateGraphRules(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData, configs models.RuleConfigs) ([]models.RuleEvaluation, error) {
	device := models.RuleEvaluation{RuleID: 6, Status: models.RuleSkipped, Reason: "Нет данных об устройстве перевода"}
	if config := configs[5]; config.Enabled && attempt.DeviceID != nil {
		users, err := repositories.CountDeviceUsers(ctx, tx, attempt, config.Integer("windowSeconds"))
		if err != nil {
			return nil, err
		}
		device.Status = models.RuleNotTriggered
		device.Reason = fmt.Sprintf("Клиентов с таким устройством: %d из %d", users, config.Integer("minUsers"))
		if users >= config.Integer("minUsers") {
			device.Status, device.Score = models.RuleTriggered, config.Points["triggered"]
		}
	}

	cycle := models.RuleEvaluation{RuleID: 7, Status: models.RuleNotTriggered, Reason: "Цикл из четырёх переводов между разными клиентами не найден"}
	if config := configs[6]; config.Enabled && attempt.SenderID != attempt.RecipientID {
		edges, err := repositories.ReadCycleEdges(ctx, tx, attempt, config.Integer("windowSeconds"))
		if err != nil {
			return nil, err
		}
		if closesTransferCycle(attempt, edges, config.Parameters["maxAmountRatio"]) {
			cycle.Status, cycle.Score, cycle.Reason = models.RuleTriggered, config.Points["triggered"], "Попытка замыкает цикл из четырёх переводов с близкими суммами"
		}
	}
	return []models.RuleEvaluation{device, cycle}, nil
}

func closesTransferCycle(attempt kafka.FraudCheckRequestedData, edges []repositories.TransferEdge, maxRatio decimal.Decimal) bool {
	graph := make(map[string][]repositories.TransferEdge)
	for _, edge := range edges {
		graph[edge.SenderID] = append(graph[edge.SenderID], edge)
	}

	visited := map[string]bool{attempt.RecipientID: true}
	var search func(string, int, decimal.Decimal, decimal.Decimal) bool
	search = func(owner string, remaining int, minAmount, maxAmount decimal.Decimal) bool {
		for _, edge := range graph[owner] {
			next := edge.RecipientID
			if visited[next] || (next == attempt.SenderID) != (remaining == 1) {
				continue
			}
			minimum := decimal.Min(minAmount, edge.Amount)
			maximum := decimal.Max(maxAmount, edge.Amount)
			if maximum.GreaterThan(minimum.Mul(maxRatio)) {
				continue
			}
			if remaining == 1 {
				return true
			}

			visited[next] = true
			if search(next, remaining-1, minimum, maximum) {
				return true
			}
			delete(visited, next)
		}
		return false
	}
	return search(attempt.RecipientID, 3, attempt.Amount, attempt.Amount)
}
