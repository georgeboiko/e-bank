package services

import (
	"context"
	"slices"
	"time"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func EvaluateTransferRules(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData) ([]models.RuleEvaluation, error) {
	facts, err := repositories.ReadTransferRuleFacts(ctx, tx, attempt)
	if err != nil {
		return nil, err
	}
	amounts, err := repositories.ReadApprovedTransferAmounts(ctx, tx, attempt)
	if err != nil {
		return nil, err
	}

	frequency := models.RuleEvaluation{RuleID: 1, Status: models.RuleNotTriggered, Reason: "Недостаточно попыток перевода в заданных окнах"}
	switch {
	case facts.LongAttempts >= 6:
		frequency.Status, frequency.Score, frequency.Reason = models.RuleTriggered, 30, "Не менее шести попыток перевода за десять минут"
	case facts.ShortAttempts >= 3:
		frequency.Status, frequency.Score, frequency.Reason = models.RuleTriggered, 15, "Не менее трёх попыток перевода за две минуты"
	}

	recipients := models.RuleEvaluation{RuleID: 4, Status: models.RuleNotTriggered, Reason: "Менее шести новых получателей за десять минут"}
	if facts.NewRecipients >= 6 {
		recipients.Status, recipients.Score, recipients.Reason = models.RuleTriggered, 25, "Не менее шести новых получателей за десять минут"
	}

	senders := models.RuleEvaluation{RuleID: 5, Status: models.RuleNotTriggered, Reason: "Менее шести разных отправителей на счёт за час"}
	if facts.SendersToAccount >= 6 {
		senders.Status, senders.Score, senders.Reason = models.RuleTriggered, 30, "Не менее шести разных отправителей на счёт за час"
	}

	return []models.RuleEvaluation{
		frequency,
		evaluateTransferAmount(attempt.Amount, amounts),
		evaluateTransferDevice(attempt.CreatedAt, facts.FirstDeviceUse, facts.FirstOtherDeviceUse),
		recipients,
		senders,
	}, nil
}

func evaluateTransferAmount(amount decimal.Decimal, history []decimal.Decimal) models.RuleEvaluation {
	if len(history) == 0 {
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleSkipped, Reason: "Нет автоматически одобренных попыток за последние 30 дней"}
	}

	median := transferMedian(history)
	deviations := make([]decimal.Decimal, len(history))
	for i, previous := range history {
		deviations[i] = previous.Sub(median).Abs()
	}
	mad := transferMedian(deviations)

	strong := decimal.Max(decimal.NewFromInt(10000), median.Mul(decimal.NewFromInt(4)), median.Add(mad.Mul(decimal.NewFromInt(6))))
	weak := decimal.Max(decimal.NewFromInt(2000), median.Mul(decimal.NewFromInt(3)), median.Add(mad.Mul(decimal.NewFromInt(4))))

	switch {
	case amount.GreaterThan(strong):
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleTriggered, Score: 30, Reason: "Сумма превышает сильный порог по истории клиента"}
	case amount.GreaterThan(weak):
		return models.RuleEvaluation{RuleID: 2, Status: models.RuleTriggered, Score: 15, Reason: "Сумма превышает слабый порог по истории клиента"}
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

func evaluateTransferDevice(createdAt time.Time, firstUse, firstOtherUse *time.Time) models.RuleEvaluation {
	if firstUse == nil {
		return models.RuleEvaluation{RuleID: 3, Status: models.RuleSkipped, Reason: "Нет данных об устройстве перевода"}
	}
	if firstOtherUse == nil || !firstOtherUse.Before(*firstUse) {
		return models.RuleEvaluation{RuleID: 3, Status: models.RuleNotTriggered, Reason: "Первоначальное устройство клиента"}
	}
	if createdAt.Sub(*firstUse) < 24*time.Hour {
		return models.RuleEvaluation{RuleID: 3, Status: models.RuleTriggered, Score: 15, Reason: "Новое устройство в первые 24 часа"}
	}
	return models.RuleEvaluation{RuleID: 3, Status: models.RuleNotTriggered, Reason: "Устройство используется не менее 24 часов"}
}
