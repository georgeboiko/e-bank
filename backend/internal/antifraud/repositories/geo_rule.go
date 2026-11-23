package repositories

import (
	"context"
	"fmt"
	"math"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func EvaluateGeoRule(ctx context.Context, tx pgx.Tx, data kafka.FraudCheckRequestedData, config models.RuleConfig) (models.RuleEvaluation, error) {
	var transferLatitude, transferLongitude, loginLatitude, loginLongitude *float64
	err := tx.QueryRow(ctx, `
		SELECT a.latitude::double precision, a.longitude::double precision,
		       l.latitude::double precision, l.longitude::double precision
		FROM transfer_attempt a
		LEFT JOIN LATERAL (
		    SELECT latitude, longitude
		    FROM login_event
		    WHERE user_id = $2
		      AND occurred_at < a.created_at
		      AND occurred_at >= a.created_at - $3::bigint * interval '1 second'
		    ORDER BY occurred_at DESC, id DESC
		    LIMIT 1
		) l ON true
		WHERE a.id = $1
	`, data.TransferID, data.SenderID, config.Integer("windowSeconds")).Scan(
		&transferLatitude, &transferLongitude, &loginLatitude, &loginLongitude,
	)
	if err != nil {
		return models.RuleEvaluation{}, fmt.Errorf("read R8 geolocation: %w", err)
	}

	result := models.RuleEvaluation{RuleID: 8, Status: models.RuleSkipped}
	if transferLatitude == nil || transferLongitude == nil {
		result.Reason = "Нет геолокации перевода"
		return result, nil
	}
	if loginLatitude == nil || loginLongitude == nil {
		result.Reason = "Нет геолокации последнего входа в заданном окне"
		return result, nil
	}

	distance := distanceKM(*transferLatitude, *transferLongitude, *loginLatitude, *loginLongitude)
	result.Status = models.RuleNotTriggered
	if decimal.NewFromFloat(distance).GreaterThanOrEqual(config.Parameters["minDistanceKm"]) {
		result.Status = models.RuleTriggered
		result.Score = config.Points["triggered"]
	}
	result.Reason = fmt.Sprintf("От последнего входа до перевода %.1f км", distance)
	return result, nil
}

func distanceKM(latitude, longitude, otherLatitude, otherLongitude float64) float64 {
	const earthRadiusKM = 6371
	const radians = math.Pi / 180

	deltaLatitude := (otherLatitude - latitude) * radians
	deltaLongitude := (otherLongitude - longitude) * radians
	sinLatitude := math.Sin(deltaLatitude / 2)
	sinLongitude := math.Sin(deltaLongitude / 2)
	a := sinLatitude*sinLatitude +
		math.Cos(latitude*radians)*math.Cos(otherLatitude*radians)*sinLongitude*sinLongitude
	return 2 * earthRadiusKM * math.Asin(math.Sqrt(min(1, a)))
}
