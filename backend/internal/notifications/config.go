package notifications

import "os"

// Config holds notification delivery environment settings.
type Config struct {
	FirebaseCredentialsPath string
	FirebaseCredentialsJSON string // base64-encoded service account JSON
	FirebaseProjectID       string
	EmailProvider           string // stub | resend | ses
	ResendAPIKey            string
	EmailFrom               string
	AWSRegion               string
	ResendWebhookSecret     string
	Environment             string
}

// LoadConfig reads notification settings from the environment.
func LoadConfig() Config {
	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		from = "noreply@studyapp.in"
	}
	provider := os.Getenv("EMAIL_PROVIDER")
	if provider == "" {
		provider = "stub"
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-south-1"
	}
	return Config{
		FirebaseCredentialsPath: os.Getenv("FIREBASE_CREDENTIALS_PATH"),
		FirebaseCredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
		FirebaseProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
		EmailProvider:           provider,
		ResendAPIKey:            os.Getenv("RESEND_API_KEY"),
		EmailFrom:               from,
		AWSRegion:               region,
		ResendWebhookSecret:     os.Getenv("RESEND_WEBHOOK_SECRET"),
		Environment:             os.Getenv("ENVIRONMENT"),
	}
}

func (c Config) IsDevelopment() bool {
	return c.Environment == "" || c.Environment == "development"
}

func (c Config) EmailConfig() EmailConfig {
	return EmailConfig{
		Provider:     c.EmailProvider,
		ResendAPIKey: c.ResendAPIKey,
		From:         c.EmailFrom,
		AWSRegion:    c.AWSRegion,
	}
}
