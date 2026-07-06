package admin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// cacheKeyPrefix mirrors scan.CacheKeyPrefix; kept local to avoid import cycles.
const cacheKeyPrefix = "quiz:hash:"

type Service struct {
	repo   Repository
	cache  *redis.Client
	logger *slog.Logger
}

func NewService(repo Repository, cache *redis.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, cache: cache, logger: logger}
}

func clampPagination(page, limit int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit, (page - 1) * limit
}

func (s *Service) GetUsers(ctx context.Context, search string, page, limit int) (UsersPage, error) {
	page, limit, offset := clampPagination(page, limit)
	users, total, err := s.repo.ListUsers(ctx, search, limit, offset)
	if err != nil {
		return UsersPage{}, err
	}
	s.enrichScanCounts(ctx, users)
	return UsersPage{Items: users, Total: total, Page: page, Limit: limit}, nil
}

func (s *Service) enrichScanCounts(ctx context.Context, users []AdminUser) {
	if s.cache == nil || len(users) == 0 {
		return
	}
	date := time.Now().Format("2006-01-02")
	for i := range users {
		key := fmt.Sprintf("scans:%d:%s", users[i].ID, date)
		if val, err := s.cache.Get(ctx, key).Int(); err == nil {
			users[i].ScanCountToday = val
		}
	}
}

func (s *Service) SuspendUser(ctx context.Context, adminID, userID int64, reason string) error {
	if err := s.repo.SetUserStatus(ctx, userID, "suspended"); err != nil {
		return err
	}
	target := fmt.Sprintf("%d", userID)
	_ = s.repo.LogAdminAction(ctx, adminID, "user.suspend", target, reason)
	_ = s.repo.LogAudit(ctx, adminID, "user.suspend", "user", target)
	return nil
}

func (s *Service) UnsuspendUser(ctx context.Context, adminID, userID int64) error {
	if err := s.repo.SetUserStatus(ctx, userID, "active"); err != nil {
		return err
	}
	target := fmt.Sprintf("%d", userID)
	_ = s.repo.LogAdminAction(ctx, adminID, "user.unsuspend", target, "")
	_ = s.repo.LogAudit(ctx, adminID, "user.unsuspend", "user", target)
	return nil
}

func (s *Service) GetJobs(ctx context.Context, status string, page, limit int) (JobsPage, error) {
	page, limit, offset := clampPagination(page, limit)
	jobs, total, err := s.repo.ListJobs(ctx, status, limit, offset)
	if err != nil {
		return JobsPage{}, err
	}
	return JobsPage{Items: jobs, Total: total, Page: page, Limit: limit}, nil
}

func (s *Service) RetryJob(ctx context.Context, adminID, jobID int64) (bool, error) {
	ok, err := s.repo.RetryJob(ctx, jobID)
	if err != nil {
		return false, err
	}
	if ok {
		target := fmt.Sprintf("%d", jobID)
		_ = s.repo.LogAdminAction(ctx, adminID, "job.retry", target, "")
		_ = s.repo.LogAudit(ctx, adminID, "job.retry", "scan_job", target)
	}
	return ok, nil
}

func (s *Service) GetAICosts(ctx context.Context, from, to time.Time) (AICostSummary, error) {
	rows, err := s.repo.AICosts(ctx, from, to)
	if err != nil {
		return AICostSummary{}, err
	}
	summary := AICostSummary{
		Rows: rows,
		From: from.Format("2006-01-02"),
		To:   to.Format("2006-01-02"),
	}
	for _, row := range rows {
		summary.TotalCost += row.TotalCost
		summary.TotalRequests += row.RequestCount
		summary.TotalTokens += row.TotalTokens
	}
	if summary.TotalRequests > 0 {
		summary.AvgCost = summary.TotalCost / float64(summary.TotalRequests)
	}
	if summary.Rows == nil {
		summary.Rows = []AICostRow{}
	}
	return summary, nil
}

func (s *Service) GetAuditLogs(ctx context.Context, actorUserID int64, page, limit int) (AuditLogPage, error) {
	page, limit, offset := clampPagination(page, limit)
	if limit > 100 {
		limit = 100
	}
	logs, total, err := s.repo.ListAuditLogs(ctx, actorUserID, limit, offset)
	if err != nil {
		return AuditLogPage{}, err
	}
	if logs == nil {
		logs = []AuditLogEntry{}
	}
	return AuditLogPage{Items: logs, Total: total, Page: page, Limit: limit}, nil
}

func (s *Service) GetContentFlags(ctx context.Context, status string, page, limit int) (ContentFlagsPage, error) {
	page, limit, offset := clampPagination(page, limit)
	flags, total, err := s.repo.ListContentFlags(ctx, status, limit, offset)
	if err != nil {
		return ContentFlagsPage{}, err
	}
	if flags == nil {
		flags = []ContentFlag{}
	}
	return ContentFlagsPage{Items: flags, Total: total, Page: page, Limit: limit}, nil
}

func (s *Service) ResolveContentFlag(ctx context.Context, adminID, flagID int64, action, reason string) error {
	status := "approved"
	if action == "removed" {
		status = "removed"
	}
	contentHash, err := s.repo.ResolveContentFlag(ctx, flagID, status, reason, adminID)
	if err != nil {
		return err
	}
	if status == "removed" && contentHash != "" {
		if err := s.repo.DeleteContentCache(ctx, contentHash); err != nil {
			s.logger.Error("failed to delete content cache", "content_hash", contentHash, "error", err)
		}
		if s.cache != nil {
			if err := s.cache.Del(ctx, cacheKeyPrefix+contentHash).Err(); err != nil {
				s.logger.Warn("failed to delete valkey cache key", "content_hash", contentHash, "error", err)
			}
		}
	}
	target := fmt.Sprintf("%d", flagID)
	_ = s.repo.LogAdminAction(ctx, adminID, "content_flag."+status, target, reason)
	_ = s.repo.LogAudit(ctx, adminID, "content_flag."+status, "content_flag", target)
	return nil
}
