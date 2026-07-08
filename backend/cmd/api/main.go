package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"studyapp/backend/internal/admin"
	"studyapp/backend/internal/auth"
	"studyapp/backend/internal/billing"
	"studyapp/backend/internal/bootstrap"
	"studyapp/backend/internal/careergoals"
	apierrors "studyapp/backend/internal/common/errors"
	"studyapp/backend/internal/common/config"
	"studyapp/backend/internal/common/health"
	"studyapp/backend/internal/common/metrics"
	custommw "studyapp/backend/internal/common/middleware"
	"studyapp/backend/internal/featureflags"
	"studyapp/backend/internal/notifications"
	"studyapp/backend/internal/quiz"
	"studyapp/backend/internal/quiz/ai"
	"studyapp/backend/internal/scan"
	"studyapp/backend/internal/scan/ocr"
	"studyapp/backend/internal/scan/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal("Config invalid: ", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	cache := redis.NewClient(&redis.Options{
		Addr: cfg.ValkeyAddr,
	})
	defer cache.Close()
	if err := cache.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to cache (valkey/memurai): %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ocrCfg := ocr.LoadOCRConfig()
	ocrProvider, err := ocr.NewProvider(ocrCfg)
	if err != nil {
		log.Fatalf("failed to init OCR provider: %v", err)
	}

	aiCfg := ai.LoadConfig()
	aiGenerator, err := ai.NewGenerator(aiCfg)
	if err != nil {
		log.Fatalf("failed to init AI generator: %v", err)
	}

	var visionGen ai.VisionGenerator
	useGeminiVision := ocrCfg.Provider == ocr.ProviderGeminiVision
	if useGeminiVision {
		visionGen, err = ai.NewGeminiVisionGenerator(aiCfg)
		if err != nil {
			log.Fatalf("failed to init Gemini Vision: %v", err)
		}
		logger.Info("gemini vision scan pipeline enabled", "model", visionGen.ModelName())
	}

	// --- Notifications module wiring (email needed for auth password reset) ---
	notificationsRepo := notifications.NewRepository(pool)
	notificationsLogger := notifications.StdLogger{}
	notifCfg := notifications.LoadConfig()

	var fcmClient *notifications.FCMClient
	if notifCfg.FirebaseCredentialsPath != "" || notifCfg.FirebaseCredentialsJSON != "" {
		client, fcmErr := notifications.NewFCMClient(notifCfg.FirebaseCredentialsPath)
		if fcmErr != nil {
			logger.Warn("FCM client init failed; push delivery disabled", "error", fcmErr)
		} else {
			fcmClient = client
			logger.Info("FCM client initialized")
		}
	} else {
		logger.Info("FCM credentials not configured; push delivery in stub mode")
	}

	emailClient, emailErr := notifications.NewEmailClient(ctx, notifCfg.EmailConfig(), logger)
	if emailErr != nil {
		log.Fatalf("failed to init email client: %v", emailErr)
	}

	notifWorker := notifications.NewNotificationWorker(pool, cache, fcmClient, emailClient, notificationsRepo, logger)

	// --- Auth module wiring ---
	authRepo := auth.NewPostgresRepository(pool)
	authService := auth.NewAuthService(authRepo, cfg.JWTSecret).
		WithPasswordResetMailer(auth.NewPasswordResetMailer(emailClient, cfg.FrontendURL)).
		WithRegistrationNotifier(notifications.NewRegistrationNotifier(notifWorker))
	authHandler := auth.NewHandler(authService, pool)

	// --- Feature flags module wiring ---
	ffRepo := featureflags.NewPostgresRepository(pool)
	ffService := featureflags.NewService(ffRepo)
	ffHandler := featureflags.NewHandler(ffService)

	bootstrap.Run(ctx, pool, ffService, bootstrap.LoadOptions(cfg.Environment), logger)

	// --- Billing module wiring ---
	billingCfg := billing.LoadConfig()
	if err := billingCfg.Validate(); err != nil {
		log.Fatal("Billing config invalid: ", err)
	}
	billingRepo := billing.NewRepository(pool)
	billingService := billing.NewService(billingRepo, pool, cache, billingCfg)
	billingHandler := billing.NewHandler(billingService, pool)

	// --- Scan module wiring ---
	scanRepo := scan.NewRepository(pool)
	scanWorkerRepo := scan.NewWorkerRepository(scanRepo, pool)
	scanStore, err := storage.NewClient()
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	scanService := scan.NewService(scanRepo, cache, scanStore)
	chunkUploader := scan.NewChunkUploadHandler(scanRepo, scanStore)
	cacheService := scan.NewCacheService(cache, pool, logger)
	if useGeminiVision {
		if n, err := cacheService.PurgeAll(ctx); err != nil {
			logger.Warn("failed to purge stale quiz cache on vision startup", "error", err)
		} else if n > 0 {
			logger.Info("purged stale stub quiz cache entries", "count", n)
		}
	}
	scanHandler := scan.NewHandler(scanService).WithChunkUpload(chunkUploader).WithCacheService(cacheService).WithBilling(billingService)

	// --- Notifications services (handler registered after routes setup) ---
	notificationsEmailService := notifications.NewEmailService(notificationsLogger, notificationsRepo, emailClient)
	notificationsFCMService := notifications.NewFCMService(notificationsLogger, notificationsRepo, fcmClient)
	notificationsService := notifications.NewService(notificationsRepo, notificationsFCMService, notificationsEmailService, notificationsLogger)
	notificationsHandler := notifications.NewHandler(
		notificationsService,
		notificationsLogger,
		notifications.WithResendWebhookSecret(notifCfg.ResendWebhookSecret),
		notifications.WithDevelopmentMode(notifCfg.IsDevelopment()),
	)

	billingService.SetPaymentNotifier(notifications.NewBillingNotifier(notifWorker))
	scanNotifier := notifications.NewScanNotifier(notifWorker, pool)

	go notifWorker.Run(ctx)

	scanWorker := scan.NewWorker(
		scanWorkerRepo,
		pool,
		cache,
		scanStore,
		ocrProvider,
		aiGenerator,
		visionGen,
		scanNotifier,
		logger,
		cacheService,
		scan.WorkerConfig{
			MaxPagesPerJob:  ocrCfg.MaxPagesPerJob,
			MinConfidence:   ocrCfg.MinConfidence,
			AIQuestionCount: aiCfg.QuestionCount,
			UseGeminiVision: useGeminiVision,
		},
	)

	// --- Quiz module wiring ---
	quizService := quiz.NewService(pool, logger).WithCache(cache).WithExplainer(ai.NewExplainer(aiCfg))
	quizHandler := quiz.NewHandler(quizService, logger)
	go func() {
		if err := scanWorker.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("scan worker stopped", "error", err)
		}
	}()
	go scan.RunCleanupWorker(ctx, pool, scanStore, cacheService, logger)

	// --- Admin module wiring ---
	adminRepo := admin.NewRepository(pool)
	adminService := admin.NewService(adminRepo, cache, logger)
	adminHandler := admin.NewHandler(adminService, pool)

	// --- Career goals module wiring (behind career_goals_module flag) ---
	cgHandler := careergoals.NewHandler(pool, cache, ffService, logger)

	metricRegistry := metrics.Init()
	prober := &health.Prober{
		Pool:            pool,
		Cache:           cache,
		Storage:         scanStore,
		OCRProviderName: func() string {
			if useGeminiVision && visionGen != nil {
				return visionGen.ProviderName()
			}
			return ocrProvider.Name()
		}(),
		AIProviderName:  aiGenerator.ProviderName(),
		Version:         "1.0.0",
	}

	rateLimiter := custommw.NewRateLimiter(cache, logger)
	allowedOrigins := custommw.LoadAllowedOrigins(cfg.Environment)
	isProduction := cfg.Environment == "production"

	const (
		maxRegularBody = 1 << 20  // 1MB
		maxUploadBody  = 11 << 20 // 11MB
	)

	r := chi.NewRouter()
	r.Use(custommw.SecurityHeaders(isProduction))
	r.Use(custommw.TraceIDMiddleware)
	r.Use(custommw.CORSMiddleware(allowedOrigins))
	r.Use(metrics.HTTPMiddleware())

	r.Get("/health", prober.Handler().ServeHTTP)
	r.Get("/ready", prober.ReadyHandler().ServeHTTP)
	r.Handle("/metrics", custommw.MetricsAllowlist(metrics.Handler(metricRegistry)))

	if localStore, ok := scanStore.(*storage.LocalClient); ok {
		r.Get("/dev/storage/*", func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimPrefix(r.URL.Path, "/dev/storage/")
			if key == "" {
				apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "missing object key", nil)
				return
			}
			unescaped, err := url.PathUnescape(key)
			if err != nil {
				apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid object key", nil)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			if err := localStore.ServeObject(w, unescaped); err != nil {
				apierrors.WriteNotFound(w, "object")
			}
		})
	}

	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFile(w, r, "docs/openapi.yaml")
	})

	r.Group(func(r chi.Router) {
		r.Use(custommw.MaxBodySize(maxRegularBody))
		r.Use(rateLimiter.ByIP(20, time.Minute, "auth"))
		authHandler.RegisterRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(custommw.MaxBodySize(maxRegularBody))
		billingHandler.RegisterPublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(custommw.MaxBodySize(maxRegularBody))
		notificationsHandler.RegisterPublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(custommw.MaxBodySize(maxRegularBody))
		r.Get("/goals", cgHandler.ListGoals)
	})

	r.Group(func(r chi.Router) {
		r.Use(custommw.RequireAuth)
		r.Use(custommw.MaxBodySize(maxRegularBody))
		r.Post("/auth/change-password", authHandler.ChangePassword)
		r.Get("/auth/me", authHandler.GetMe)
		r.Get("/me/features", ffHandler.GetMyFeatures)
		quizHandler.RegisterRoutes(r)
		notificationsHandler.RegisterRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(rateLimiter.ByUser(30, time.Minute, "scan"))
			scanHandler.RegisterRoutes(r)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimiter.ByUser(30, time.Minute, "scan"))
			r.Use(custommw.MaxBodySize(maxUploadBody))
			scanHandler.RegisterUploadRoutes(r)
		})

		r.Group(func(r chi.Router) {
			r.Use(rateLimiter.ByUser(10, time.Minute, "billing"))
			billingHandler.RegisterAuthRoutes(r)
			if cfg.Environment == "development" {
				billingHandler.RegisterDevRoutes(r)
			}
		})
	})

	// Career goals: all endpoints below require auth AND the career_goals_module
	// flag. When the flag is OFF these return 403 feature_not_available.
	r.Group(func(r chi.Router) {
		r.Use(custommw.RequireAuth)
		r.Use(custommw.MaxBodySize(maxRegularBody))
		r.Use(careergoals.RequireCareerGoalsFlag(ffService))
		r.Post("/goals/select", cgHandler.SelectGoal)
		r.Get("/goals/my", cgHandler.GetMyGoal)
		r.Delete("/goals/my", cgHandler.AbandonGoal)
		r.Get("/goals/my/practice/today", cgHandler.GetTodayPractice)
		r.Post("/goals/my/practice/{setId}/submit", cgHandler.SubmitPractice)
		r.Get("/goals/my/practice/history", cgHandler.GetPracticeHistory)
		r.Get("/goals/my/skills", cgHandler.GetSkillGaps)
	})

	r.Group(func(r chi.Router) {
		r.Use(custommw.RequireAuth)
		r.Use(custommw.RequireAdmin)
		r.Use(custommw.MaxBodySize(maxRegularBody))
		r.Use(rateLimiter.ByUser(100, time.Minute, "admin"))
		r.Get("/admin/features", ffHandler.AdminListFlags)
		r.Post("/admin/features", ffHandler.AdminSetFlag)
		scanHandler.RegisterAdminRoutes(r)
		adminHandler.RegisterRoutes(r)
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("studyapp backend starting on :%s (env=%s, ocr=%s)", cfg.Port, cfg.Environment, ocrProvider.Name())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("studyapp backend stopped")
}
