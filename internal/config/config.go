package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv      string
	ServiceName string
	HTTPPort    string

	Postgres PostgresConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	JWT      JWTConfig

	LeadServiceURL         string
	CustomerServiceURL     string
	NotificationServiceURL string
	ReadinessTimeout       time.Duration
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
}

type JWTConfig struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

func Load(serviceName, defaultPort string) Config {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v, serviceName, defaultPort)

	return Config{
		AppEnv:      v.GetString("app.env"),
		ServiceName: serviceName,
		HTTPPort:    v.GetString("http.port"),
		Postgres: PostgresConfig{
			Host:     v.GetString("postgres.host"),
			Port:     v.GetInt("postgres.port"),
			User:     v.GetString("postgres.user"),
			Password: v.GetString("postgres.password"),
			Database: v.GetString("postgres.db"),
			SSLMode:  v.GetString("postgres.sslmode"),
		},
		Redis: RedisConfig{
			Host:     v.GetString("redis.host"),
			Port:     v.GetInt("redis.port"),
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		Kafka: KafkaConfig{
			Brokers: splitCSV(v.GetString("kafka.brokers")),
		},
		JWT: JWTConfig{
			Secret: v.GetString("jwt.secret"),
			Issuer: v.GetString("jwt.issuer"),
			TTL:    v.GetDuration("jwt.ttl"),
		},
		LeadServiceURL:         strings.TrimRight(v.GetString("lead.service.url"), "/"),
		CustomerServiceURL:     strings.TrimRight(v.GetString("customer.service.url"), "/"),
		NotificationServiceURL: strings.TrimRight(v.GetString("notification.service.url"), "/"),
		ReadinessTimeout:       v.GetDuration("readiness.timeout"),
	}
}

func setDefaults(v *viper.Viper, serviceName, defaultPort string) {
	v.SetDefault("app.env", "local")
	v.SetDefault("http.port", defaultPort)
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.user", "crm")
	v.SetDefault("postgres.password", "crm")
	v.SetDefault("postgres.db", "crm")
	v.SetDefault("postgres.sslmode", "disable")
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("kafka.brokers", "localhost:9092")
	v.SetDefault("jwt.secret", "change-me")
	v.SetDefault("jwt.issuer", "event-driven-crm")
	v.SetDefault("jwt.ttl", 24*time.Hour)
	v.SetDefault("lead.service.url", "http://localhost:8081")
	v.SetDefault("customer.service.url", "http://localhost:8082")
	v.SetDefault("notification.service.url", "http://localhost:8083")
	v.SetDefault("readiness.timeout", 2*time.Second)
	v.SetDefault("service.name", serviceName)
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
