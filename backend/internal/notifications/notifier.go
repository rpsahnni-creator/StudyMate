package notifications

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScanNotifier enqueues quiz-ready notifications for the scan worker.
type ScanNotifier struct {
	worker *NotificationWorker
	db     *pgxpool.Pool
}

// NewScanNotifier creates a notifier that implements scan.Notifier.
func NewScanNotifier(worker *NotificationWorker, db *pgxpool.Pool) *ScanNotifier {
	return &ScanNotifier{worker: worker, db: db}
}

// QuizReady enqueues push + email for a completed scan quiz.
func (n *ScanNotifier) QuizReady(ctx context.Context, userID, jobID, quizID int64) error {
	if n == nil || n.worker == nil {
		return nil
	}

	subject, count := n.lookupQuizMeta(ctx, jobID, quizID)

	return n.worker.Enqueue(ctx, QueueJob{
		UserID:     userID,
		TemplateID: TmplQuizReady,
		Data: map[string]string{
			"subject": subject,
			"count":   strconv.Itoa(count),
			"quizUrl": fmt.Sprintf("/quiz/%d", quizID),
		},
		Channels: []string{ChannelPush, ChannelEmail},
	})
}

func (n *ScanNotifier) lookupQuizMeta(ctx context.Context, jobID, quizID int64) (string, int) {
	subject := "Study"
	count := 0

	if n.db == nil {
		return subject, count
	}

	err := n.db.QueryRow(ctx, `
		SELECT COALESCE(sj.mode, 'Study'), COALESCE(q.total_questions, 0)
		FROM scan_jobs sj
		LEFT JOIN quizzes q ON q.id = $2
		WHERE sj.id = $1
	`, jobID, quizID).Scan(&subject, &count)
	if err != nil {
		return "Study", 0
	}
	return subject, count
}

// BillingNotifier enqueues payment notifications for billing webhooks.
type BillingNotifier struct {
	worker *NotificationWorker
}

// NewBillingNotifier creates a notifier for billing events.
func NewBillingNotifier(worker *NotificationWorker) *BillingNotifier {
	return &BillingNotifier{worker: worker}
}

// PaymentCaptured enqueues payment success notifications.
func (n *BillingNotifier) PaymentCaptured(ctx context.Context, userID int64, planName, expiresAt string) error {
	if n == nil || n.worker == nil {
		return nil
	}
	return n.worker.Enqueue(ctx, QueueJob{
		UserID:     userID,
		TemplateID: TmplPaymentSuccess,
		Data: map[string]string{
			"plan":    planName,
			"expires": expiresAt,
		},
		Channels: []string{ChannelPush, ChannelEmail},
	})
}

// PaymentFailed enqueues payment failure notifications.
func (n *BillingNotifier) PaymentFailed(ctx context.Context, userID int64) error {
	if n == nil || n.worker == nil {
		return nil
	}
	return n.worker.Enqueue(ctx, QueueJob{
		UserID:     userID,
		TemplateID: TmplPaymentFailed,
		Data:       map[string]string{},
		Channels:   []string{ChannelPush, ChannelEmail},
	})
}
