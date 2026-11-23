package services

import (
	"context"
	"log/slog"
	"time"

	"e-bank/backend/internal/money/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

const transferFinalizationTimeout = 5 * time.Second

func RunTransferFinalizer(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for ctx.Err() == nil {
		finalizePendingTransfers(ctx, logger, pool)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func finalizePendingTransfers(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) {
	readCtx, cancel := context.WithTimeout(ctx, transferFinalizationTimeout)
	lastID, err := repositories.LastPendingFinalizationID(readCtx, pool)
	cancel()
	if err != nil {
		if ctx.Err() == nil {
			logger.Warn("Could not read pending Money transfers", "error", err)
		}
		return
	}

	// Bound this pass so new work cannot indefinitely delay retries of earlier transfers.
	for afterID := int32(0); afterID < lastID && ctx.Err() == nil; {
		attemptCtx, cancel := context.WithTimeout(ctx, transferFinalizationTimeout)
		id, err := repositories.FinalizeNextTransfer(attemptCtx, pool, afterID, lastID)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Warn("Money transfer finalization failed", "transferId", id, "error", err)
		}
		if id == 0 {
			return
		}
		afterID = id
	}
}
