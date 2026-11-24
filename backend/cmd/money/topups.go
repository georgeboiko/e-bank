package main

import (
	"errors"
	"os"
	"strconv"

	"e-bank/backend/internal/infrastructure/yookassa"
	"e-bank/backend/internal/money/repositories"
	"e-bank/backend/internal/money/services"

	"github.com/jackc/pgx/v5/pgxpool"
)

func configureTopups(pool *pgxpool.Pool) (*services.TopupService, error) {
	repository := repositories.NewTopupRepository(pool)
	shopID := os.Getenv("YOOKASSA_SHOP_ID")
	secret := os.Getenv("YOOKASSA_SECRET_KEY")
	returnURL := os.Getenv("YOOKASSA_RETURN_URL")
	if shopID == "" && secret == "" && returnURL == "" {
		return services.NewTopupService(repository, nil, 0), nil
	}
	technicalID, err := strconv.ParseInt(os.Getenv("MONEY_TOPUP_TECHNICAL_ACCOUNT_ID"), 10, 32)
	if err != nil || technicalID < 1 {
		return nil, errors.New("MONEY_TOPUP_TECHNICAL_ACCOUNT_ID must identify the technical account")
	}
	client, err := yookassa.NewClient(yookassa.APIURL, shopID, secret, returnURL)
	if err != nil {
		return nil, err
	}
	return services.NewTopupService(repository, client, int32(technicalID)), nil
}
