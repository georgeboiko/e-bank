package services

import (
	"context"
	"fmt"
	"slices"
	"time"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func EvaluateTransferRules(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData, configs models.RuleConfigs) ([]models.RuleEvaluation, error) {
	facts, err := repositories.ReadTransferRuleFacts(ctx, tx, attempt, configs)
	if err != nil {
		return nil, err
	}
	amount := disabledRule(2)
	if config := configs[1]; config.Enabled {
		amounts, err := repositories.ReadApprovedTransferAmounts(ctx, tx, attempt, config.Integer("historyDays"))
		if err != nil {
			return nil, err
		}
		amount = evaluateTransferAmount(attempt.Amount, amounts, config)
	}

	frequency := models.RuleEvaluation{RuleID: 1, Status: models.RuleNotTriggered, Reason: "Недостаточно попыток перевода в заданных окнах"}
	switch {
	case facts.LongAttempts >= configs[0].Integer("longCount"):
		frequency.Status, frequency.Score, frequency.Reason = models.RuleTriggered, configs[0].Points["strong"],
			fmt.Sprintf("Попыток в длинном окне: %d из %d", facts.LongAttempts, configs[0].Integer("longCount"))
	case facts.ShortAttempts >= configs[0].Integer("shortCount"):
		frequency.Status, frequency.Score, frequency.Reason = models.RuleTriggered, configs[0].Points["weak"],
			fmt.Sprintf("Попыток в коротком окне: %d из %d", facts.ShortAttempts, configs[0].Integer("shortCount"))
	}

	recipients := models.RuleEvaluation{RuleID: 4, Status: models.RuleNotTriggered, Reason: fmt.Sprintf("Новых получателей: %d из %d", facts.NewRecipients, configs[3].Integer("minNewRecipients"))}
	if facts.NewRecipients >= configs[3].Integer("minNewRecipients") {
		recipients.Status, recipients.Score = models.RuleTriggered, configs[3].Points["triggered"]
	}

	senders := models.RuleEvaluation{RuleID: 5, Status: models.RuleNotTriggered, Reason: fmt.Sprintf("Разных отправителей на счёт: %d из %d", facts.SendersToAccount, configs[4].Integer("minSenders"))}
	if facts.SendersToAccount >= configs[4].Integer("minSenders") {
		senders.Status, senders.Score = models.RuleTriggered, configs[4].Points["triggered"]
	}

	return []models.RuleEvaluation{
		frequency,
		amount,
		evaluateTransferDevice(attempt.CreatedAt, facts.FirstDeviceUse, facts.FirstOtherDeviceUse, configs[2]),
		recipients,
		senders,
	}, nil
}

func evaluateTransferAmount(amount decimal.Decimal, history []decimal.Decimal, config models.RuleConfig) models.RuleEvaluation {
	if len(history) == 0 {
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleSkipped, Reason: "Нет автоматически одобренных попыток в заданном окне"}
	}

	median := transferMedian(history)
	deviations := make([]decimal.Decimal, len(history))
	for i, previous := range history {
		deviations[i] = previous.Sub(median).Abs()
	}
	mad := transferMedian(deviations)

	p := config.Parameters
	strong := decimal.Max(p["strongMinAmount"], median.Mul(p["strongMedianFactor"]), median.Add(mad.Mul(p["strongMadFactor"])))
	weak := decimal.Max(p["weakMinAmount"], median.Mul(p["weakMedianFactor"]), median.Add(mad.Mul(p["weakMadFactor"])))

	switch {
	case amount.GreaterThan(strong):
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleTriggered, Score: config.Points["strong"], Reason: "Сумма превышает сильный порог по истории клиента"}
	case amount.GreaterThan(weak):
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleTriggered, Score: config.Points["weak"], Reason: "Сумма превышает слабый порог по истории клиента"}
	default:
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleNotTriggered, Reason: "Сумма не превышает пороги по истории клиента"}
	}
}

func transferMedian(values []decimal.Decimal) decimal.Decimal {
	slices.SortFunc(values, decimal.Decimal.Cmp)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return values[mid-1].Add(values[mid]).Mul(decimal.New(5, -1))
}

func evaluateTransferDevice(createdAt time.Time, firstUse, firstOtherUse *time.Time, config models.RuleConfig) models.RuleEvaluation {
	if firstUse == nil {
		return models.RuleEvaluation{RuleID: 3, Status: models.RuleSkipped, Reason: "Нет данных об устройстве перевода"}
	}
	if firstOtherUse == nil || !firstOtherUse.Before(*firstUse) {
		return models.RuleEvaluation{RuleID: 3, Status: models.RuleNotTriggered, Reason: "Первоначальное устройство клиента"}
	}
	if createdAt.Sub(*firstUse) < time.Duration(config.Integer("newDeviceWindowSeconds"))*time.Second {
		return models.RuleEvaluation{RuleID: 3, Status: models.RuleTriggered, Score: config.Points["triggered"], Reason: "Первое использование нового устройства попадает в заданное окно"}
	}
	return models.RuleEvaluation{RuleID: 3, Status: models.RuleNotTriggered, Reason: "Первое использование устройства было до начала заданного окна"}
}
