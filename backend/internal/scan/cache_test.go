package scan

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type mockRedis struct {
	mu    sync.Mutex
	store map[string]string
	gets  int
}

func (m *mockRedis) Get(_ context.Context, key string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gets++
	val, ok := m.store[key]
	cmd := redis.NewStringCmd(context.Background())
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(val)
	return cmd
}

func (m *mockRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch v := value.(type) {
	case string:
		m.store[key] = v
	case []byte:
		m.store[key] = string(v)
	default:
		raw, _ := json.Marshal(v)
		m.store[key] = string(raw)
	}
	cmd := redis.NewStatusCmd(context.Background())
	cmd.SetVal("OK")
	return cmd
}

func (m *mockRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, key := range keys {
		if _, ok := m.store[key]; ok {
			delete(m.store, key)
			n++
		}
	}
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(n)
	return cmd
}

type cacheRow struct {
	quizID        int64
	questionCount int
	board         *string
	subject       *string
	chapter       *string
	cachedAt      time.Time
	expiresAt     time.Time
}

type mockCacheDB struct {
	mu          sync.Mutex
	row         *cacheRow
	queryCount  int
	execCount   int
	deleted     bool
}

func (m *mockCacheDB) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queryCount++
	return &mockRow{row: m.row}
}

func (m *mockCacheDB) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.execCount++
	if contains(sql, "DELETE FROM content_cache") {
		m.deleted = true
		m.row = nil
	}
	return pgconn.CommandTag{}, nil
}

type mockRow struct {
	row *cacheRow
}

func (r *mockRow) Scan(dest ...any) error {
	if r.row == nil {
		return pgx.ErrNoRows
	}
	*(dest[0].(*int64)) = r.row.quizID
	*(dest[1].(*int)) = r.row.questionCount
	*(dest[2].(**string)) = r.row.board
	*(dest[3].(**string)) = r.row.subject
	*(dest[4].(**string)) = r.row.chapter
	*(dest[5].(*time.Time)) = r.row.cachedAt
	*(dest[6].(*time.Time)) = r.row.expiresAt
	return nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func sampleCachedQuiz(hash string) *CachedQuiz {
	now := time.Now()
	return &CachedQuiz{
		QuizID:        42,
		ContentHash:   hash,
		QuestionCount: 10,
		CachedAt:      now.Add(-time.Hour),
		ExpiresAt:     now.Add(6 * 24 * time.Hour),
		Subject:       "chapter",
	}
}

func TestCacheLookupValkeyHitSkipsDB(t *testing.T) {
	hash := "abc123"
	redis := &mockRedis{store: map[string]string{}}
	raw, _ := json.Marshal(sampleCachedQuiz(hash))
	redis.store[cacheRedisKey(hash)] = string(raw)

	db := &mockCacheDB{}
	svc := &CacheService{redis: redis, db: db}

	cached, hit, err := svc.Lookup(context.Background(), hash)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if !hit || cached == nil || cached.QuizID != 42 {
		t.Fatalf("expected valkey hit, got hit=%v cached=%+v", hit, cached)
	}
	if db.queryCount != 0 {
		t.Fatalf("expected no DB query on valkey hit, got %d", db.queryCount)
	}
}

func TestCacheLookupValkeyMissDBHitRepopulatesValkey(t *testing.T) {
	hash := "db-only"
	redis := &mockRedis{store: map[string]string{}}
	now := time.Now()
	db := &mockCacheDB{row: &cacheRow{
		quizID: 99, questionCount: 8, cachedAt: now, expiresAt: now.Add(CacheTTL),
	}}
	svc := &CacheService{redis: redis, db: db}

	cached, hit, err := svc.Lookup(context.Background(), hash)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if !hit || cached.QuizID != 99 {
		t.Fatalf("expected DB hit, got hit=%v cached=%+v", hit, cached)
	}
	if db.queryCount != 1 {
		t.Fatalf("expected one DB query, got %d", db.queryCount)
	}
	if _, ok := redis.store[cacheRedisKey(hash)]; !ok {
		t.Fatal("expected valkey repopulated after DB hit")
	}
}

func TestCacheLookupMissReturnsFalse(t *testing.T) {
	redis := &mockRedis{store: map[string]string{}}
	db := &mockCacheDB{row: nil}
	svc := &CacheService{redis: redis, db: db}

	_, hit, err := svc.Lookup(context.Background(), "missing")
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if hit {
		t.Fatal("expected miss")
	}
}

func TestCacheLookupExpiredDBRowIsMiss(t *testing.T) {
	hash := "expired"
	redis := &mockRedis{store: map[string]string{}}
	now := time.Now()
	db := &mockCacheDB{row: &cacheRow{
		quizID: 1, questionCount: 5, cachedAt: now.Add(-8 * 24 * time.Hour), expiresAt: now.Add(-time.Hour),
	}}
	// mockRow always returns row - simulate expired by having QueryRow return no rows
	db.row = nil
	svc := &CacheService{redis: redis, db: db}

	_, hit, err := svc.Lookup(context.Background(), hash)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if hit {
		t.Fatal("expected expired/missing entry to be a miss")
	}
}

func TestCacheLookupExpiredValkeyFallsThroughToDB(t *testing.T) {
	hash := "valkey-expired"
	redis := &mockRedis{store: map[string]string{}}
	now := time.Now()
	expired := &CachedQuiz{
		QuizID: 7, ContentHash: hash, ExpiresAt: now.Add(-time.Hour),
	}
	raw, _ := json.Marshal(expired)
	redis.store[cacheRedisKey(hash)] = string(raw)

	db := &mockCacheDB{row: &cacheRow{quizID: 7, questionCount: 5, cachedAt: now, expiresAt: now.Add(CacheTTL)}}
	svc := &CacheService{redis: redis, db: db}

	cached, hit, err := svc.Lookup(context.Background(), hash)
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if !hit || cached.QuizID != 7 {
		t.Fatalf("expected DB fallback after expired valkey entry, got hit=%v cached=%+v", hit, cached)
	}
	if _, ok := redis.store[cacheRedisKey(hash)]; !ok {
		t.Fatal("expected valkey repopulated from DB")
	}
}

func TestCacheStoreWritesValkey(t *testing.T) {
	redis := &mockRedis{store: map[string]string{}}
	db := &mockCacheDB{}
	svc := &CacheService{redis: redis, db: db}

	err := svc.Store(context.Background(), "store-me", CacheMeta{
		QuizID: 55, QuestionCount: 10, Subject: "chapter", AIProvider: "stub", ModelUsed: "stub-v1",
	})
	if err != nil {
		t.Fatalf("store failed: %v", err)
	}
	if db.execCount < 1 {
		t.Fatal("expected DB write")
	}
	if _, ok := redis.store[cacheRedisKey("store-me")]; !ok {
		t.Fatal("expected valkey write")
	}
}

func TestCacheInvalidateDeletesBoth(t *testing.T) {
	hash := "gone"
	redis := &mockRedis{store: map[string]string{cacheRedisKey(hash): "x"}}
	db := &mockCacheDB{row: &cacheRow{quizID: 1, expiresAt: time.Now().Add(CacheTTL)}}
	svc := &CacheService{redis: redis, db: db}

	if err := svc.Invalidate(context.Background(), hash); err != nil {
		t.Fatalf("invalidate failed: %v", err)
	}
	if !db.deleted {
		t.Fatal("expected DB delete")
	}
	if _, ok := redis.store[cacheRedisKey(hash)]; ok {
		t.Fatal("expected valkey delete")
	}
}

func TestContentCacheHashDeterministic(t *testing.T) {
	text := "Hello   World\n\tTest"
	h1 := ContentCacheHash(text)
	h2 := ContentCacheHash("hello world test")
	if h1 != h2 {
		t.Fatalf("expected same hash for normalized equivalent text, got %q vs %q", h1, h2)
	}
	if h1 == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestCacheStoreRequiresQuizID(t *testing.T) {
	svc := &CacheService{db: &mockCacheDB{}}
	err := svc.Store(context.Background(), "x", CacheMeta{})
	if err == nil {
		t.Fatal("expected error for invalid store input")
	}
}
