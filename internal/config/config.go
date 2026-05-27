package config

import "os"

type Config struct {
	DSN            string
	RabbitURL      string
	GurenKey       string
	WebhookSiteURL string
}

func Load() *Config {
	return &Config{
		DSN:            GetEnv("DSN", "mysql:mysql@tcp(mysql:3306)/db?parseTime=true&loc=UTC"),
		RabbitURL:      GetEnv("RABBIT_URL", "amqp://rabbit:rabbit@rabbitmq:5672/rabbit"),
		GurenKey:       GetEnv("GUREN_KEY", ""),
		WebhookSiteURL: GetEnv("WEBHOOK_SITE_URL", ""),
	}
}

func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
