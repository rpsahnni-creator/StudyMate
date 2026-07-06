package billing

import "os"

// Config holds billing provider credentials from environment.
type Config struct {
	RazorpayKeyID        string
	RazorpayKeySecret    string
	RazorpayWebhookSecret string
	PayUKey              string
	PayUSalt             string
	PayUWebhookURL       string
	Environment          string
}

func LoadConfig() Config {
	return Config{
		RazorpayKeyID:         os.Getenv("RAZORPAY_KEY_ID"),
		RazorpayKeySecret:     os.Getenv("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
		PayUKey:               os.Getenv("PAYU_KEY"),
		PayUSalt:              os.Getenv("PAYU_SALT"),
		PayUWebhookURL:        os.Getenv("PAYU_WEBHOOK_URL"),
		Environment:           os.Getenv("ENVIRONMENT"),
	}
}

func (c Config) IsDevelopment() bool {
	return c.Environment == "" || c.Environment == "development"
}
