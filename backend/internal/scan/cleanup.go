package scan

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"studyapp/backend/internal/scan/storage"
)

const cleanupInterval = time.Hour

// RunCleanupWorker periodically deletes orphaned temp storage objects and expired cache rows.
func RunCleanupWorker(ctx context.Context, db *pgxpool.Pool, store storage.Client, cacheService *CacheService, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	logger.Info("scan cleanup worker started", "interval", cleanupInterval.String())

	var lastCachePurge time.Time

	for {
		select {
		case <-ctx.Done():
			logger.Info("scan cleanup worker shutting down")
			return
		case <-ticker.C:
			count, err := runCleanupOnce(ctx, db, store)
			if err != nil {
				logger.Error("scan cleanup failed", "error", err)
			} else if count > 0 {
				logger.Info("scan cleanup deleted orphaned objects", "count", count)
			}

			if cacheService != nil && time.Since(lastCachePurge) >= 7*24*time.Hour {
				purged, err := cacheService.PurgeExpired(ctx)
				if err != nil {
					logger.Error("cache purge failed", "error", err)
				} else if purged > 0 {
					logger.Info("cache purge deleted expired entries", "count", purged)
				}
				lastCachePurge = time.Now()
			}
		}
	}
}

func runCleanupOnce(ctx context.Context, db *pgxpool.Pool, store storage.Client) (int, error) {
	rows, err := db.Query(ctx, `
		SELECT id, temp_storage_key
		FROM scan_pages
		WHERE created_at < NOW() - INTERVAL '25 hours'
		  AND temp_storage_key IS NOT NULL
		  AND deleted_at IS NULL
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type orphan struct {
		id  int64
		key string
	}
	var orphans []orphan
	for rows.Next() {
		var o orphan
		if err := rows.Scan(&o.id, &o.key); err != nil {
			return 0, err
		}
		orphans = append(orphans, o)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	deleted := 0
	for _, o := range orphans {
		if err := store.DeleteObject(ctx, o.key); err != nil {
			continue
		}
		_, err := db.Exec(ctx, `
			UPDATE scan_pages SET deleted_at = NOW() WHERE id = $1
		`, o.id)
		if err != nil {
			continue
		}
		deleted++
	}
	return deleted, nil
}
