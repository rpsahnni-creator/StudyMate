package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	// CacheTTL is how long quiz cache entries live in Valkey and Postgres.
	CacheTTL = 7 * 24 * time.Hour
	// CacheKeyPrefix is the Valkey key namespace for content-hash lookups.
	CacheKeyPrefix = "quiz:hash:"
)

// CachedQuiz is a cache hit payload — never contains raw OCR text.
type CachedQuiz struct {
	QuizID        int64     `json:"quiz_id"`
	ContentHash   string    `json:"content_hash"`
	QuestionCount int       `json:"question_count"`
	CachedAt      time.Time `json:"cached_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Board         string    `json:"board,omitempty"`
	Subject       string    `json:"subject,omitempty"`
	Chapter       string    `json:"chapter,omitempty"`
}

// CacheMeta captures metadata written alongside a cache entry.
type CacheMeta struct {
	Board        string
	Subject      string
	Chapter      string
	QuizID       int64
	AIProvider   string
	ModelUsed    string
	TokensUsed   int
	GenerationMs int64
	QuestionCount int
	ScanJobID    int64
	BookID       *int64
	ChapterID    *int64
	PageNo       int
}

type redisStore interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type cacheDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// CacheService implements two-tier Valkey + Postgres content caching.
type CacheService struct {
	redis  redisStore
	db     cacheDB
	logger *slog.Logger
}

func NewCacheService(redisClient *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *CacheService {
	if logger == nil {
		logger = slog.Default()
	}
	var store redisStore
	if redisClient != nil {
		store = redisClient
	}
	var dbStore cacheDB
	if db != nil {
		dbStore = db
	}
	return &CacheService{redis: store, db: dbStore, logger: logger}
}

func cacheRedisKey(contentHash string) string {
	return CacheKeyPrefix + contentHash
}

// Lookup checks Valkey first, then Postgres. Returns hit=false on miss or expiry.
func (c *CacheService) Lookup(ctx context.Context, contentHash string) (*CachedQuiz, bool, error) {
	if contentHash == "" {
		return nil, false, nil
	}

	if c.redis != nil {
		if cached, hit, err := c.lookupRedis(ctx, contentHash); err != nil {
			return nil, false, err
		} else if hit {
			return cached, true, nil
		}
	}

	if c.db == nil {
		return nil, false, nil
	}

	return c.lookupPostgres(ctx, contentHash)
}

func (c *CacheService) lookupRedis(ctx context.Context, contentHash string) (*CachedQuiz, bool, error) {
	raw, err := c.redis.Get(ctx, cacheRedisKey(contentHash)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if raw == "" {
		return nil, false, nil
	}

	var cached CachedQuiz
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		if quizID, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && quizID > 0 {
			cached = CachedQuiz{QuizID: quizID, ContentHash: contentHash}
		} else {
			return nil, false, fmt.Errorf("decode valkey cache entry: %w", err)
		}
	}
	if !cached.ExpiresAt.IsZero() && time.Now().After(cached.ExpiresAt) {
		_ = c.redis.Del(ctx, cacheRedisKey(contentHash)).Err()
		return nil, false, nil
	}
	return &cached, true, nil
}

func (c *CacheService) lookupPostgres(ctx context.Context, contentHash string) (*CachedQuiz, bool, error) {
	var (
		quizID        int64
		questionCount int
		board         *string
		subject       *string
		chapter       *string
		cachedAt      time.Time
		expiresAt     time.Time
	)
	err := c.db.QueryRow(ctx, `
		SELECT generated_quiz_id, question_count, board, subject, chapter, created_at, expires_at
		FROM content_cache
		WHERE content_hash = $1 AND expires_at > NOW()
	`, contentHash).Scan(&quizID, &questionCount, &board, &subject, &chapter, &cachedAt, &expiresAt)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if quizID <= 0 {
		return nil, false, nil
	}

	cached := &CachedQuiz{
		QuizID:        quizID,
		ContentHash:   contentHash,
		QuestionCount: questionCount,
		CachedAt:      cachedAt,
		ExpiresAt:     expiresAt,
	}
	if board != nil {
		cached.Board = *board
	}
	if subject != nil {
		cached.Subject = *subject
	}
	if chapter != nil {
		cached.Chapter = *chapter
	}

	if c.redis != nil {
		remaining := time.Until(expiresAt)
		if remaining > 0 {
			_ = c.setRedis(ctx, contentHash, cached, remaining)
		}
	}

	_, _ = c.db.Exec(ctx, `
		UPDATE content_cache SET hit_count = hit_count + 1 WHERE content_hash = $1
	`, contentHash)

	return cached, true, nil
}

// Store persists a cache entry to Postgres and Valkey.
func (c *CacheService) Store(ctx context.Context, contentHash string, meta CacheMeta) error {
	if contentHash == "" || meta.QuizID <= 0 {
		return fmt.Errorf("invalid cache store input")
	}
	if c.db == nil {
		return fmt.Errorf("database not configured for cache store")
	}

	expiresAt := time.Now().Add(CacheTTL)
	cachedAt := time.Now()

	_, err := c.db.Exec(ctx, `
		INSERT INTO content_cache (
			book_id, chapter_id, page_no, content_hash, page_type,
			generated_quiz_id, question_count, board, subject, chapter,
			expires_at, hit_count, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 1, $12)
		ON CONFLICT (content_hash) DO UPDATE SET
			generated_quiz_id = EXCLUDED.generated_quiz_id,
			question_count = EXCLUDED.question_count,
			board = EXCLUDED.board,
			subject = EXCLUDED.subject,
			chapter = EXCLUDED.chapter,
			expires_at = EXCLUDED.expires_at,
			hit_count = content_cache.hit_count + 1
	`, meta.BookID, meta.ChapterID, meta.PageNo, contentHash, meta.Subject,
		meta.QuizID, meta.QuestionCount, nullIfEmptyStr(meta.Board), nullIfEmptyStr(meta.Subject),
		nullIfEmptyStr(meta.Chapter), expiresAt, cachedAt)
	if err != nil {
		return err
	}

	entry := CachedQuiz{
		QuizID:        meta.QuizID,
		ContentHash:   contentHash,
		QuestionCount: meta.QuestionCount,
		CachedAt:      cachedAt,
		ExpiresAt:     expiresAt,
		Board:         meta.Board,
		Subject:       meta.Subject,
		Chapter:       meta.Chapter,
	}
	if c.redis != nil {
		if err := c.setRedis(ctx, contentHash, &entry, CacheTTL); err != nil {
			c.logger.Warn("valkey cache write failed", "content_hash", contentHash, "error", err)
		}
	}

	c.logCacheEvent(ctx, contentHash, meta, false)
	return nil
}

// LogCacheHit records a cache hit in ai_generation_logs.
func (c *CacheService) LogCacheHit(ctx context.Context, contentHash string, scanJobID int64, provider string, cached *CachedQuiz) {
	if cached == nil {
		return
	}
	meta := CacheMeta{
		QuizID:        cached.QuizID,
		QuestionCount: cached.QuestionCount,
		AIProvider:    provider,
		ScanJobID:     scanJobID,
	}
	c.logCacheEvent(ctx, contentHash, meta, true)
}

func (c *CacheService) logCacheEvent(ctx context.Context, contentHash string, meta CacheMeta, cacheHit bool) {
	if c.db == nil {
		return
	}
	var jobID any
	if meta.ScanJobID > 0 {
		jobID = meta.ScanJobID
	}
	status := "success"
	if cacheHit {
		status = "cache_hit"
	}
	_, err := c.db.Exec(ctx, `
		INSERT INTO ai_generation_logs
			(scan_job_id, provider, model_name, question_count, token_usage, duration_ms, cache_hit, content_hash, cost_estimate, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, now())
	`, jobID, meta.AIProvider, meta.ModelUsed, meta.QuestionCount, meta.TokensUsed, meta.GenerationMs, cacheHit, contentHash, status)
	if err != nil {
		c.logger.Error("failed to log cache event", "content_hash", contentHash, "cache_hit", cacheHit, "error", err)
	}
}

// Invalidate removes a cache entry from Postgres and Valkey (admin use).
func (c *CacheService) Invalidate(ctx context.Context, contentHash string) error {
	if contentHash == "" {
		return fmt.Errorf("content hash is required")
	}
	if c.db != nil {
		if _, err := c.db.Exec(ctx, `DELETE FROM content_cache WHERE content_hash = $1`, contentHash); err != nil {
			return err
		}
	}
	if c.redis != nil {
		if err := c.redis.Del(ctx, cacheRedisKey(contentHash)).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CacheService) setRedis(ctx context.Context, contentHash string, entry *CachedQuiz, ttl time.Duration) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, cacheRedisKey(contentHash), raw, ttl).Err()
}

// PurgeExpired removes expired rows from Postgres (weekly cleanup).
func (c *CacheService) PurgeExpired(ctx context.Context) (int64, error) {
	if c.db == nil {
		return 0, nil
	}
	tag, err := c.db.Exec(ctx, `DELETE FROM content_cache WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func nullIfEmptyStr(value string) any {
	if value == "" {
		return nil
	}
	return value
}

const contentCacheVersion = "stub-v3"

// ContentCacheHash returns SHA256 of normalized OCR text for cache keys.
func ContentCacheHash(rawText string) string {
	return contentHashFromNormalized(contentCacheVersion + "|" + normalizeText(rawText))
}

func contentHashFromNormalized(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
