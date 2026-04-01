// Package config จัดการ configuration ของ provider-backoffice-api
//
// ความสัมพันธ์:
// - repo #9 — API เดียว serve ทั้ง admin (#10) + operator dashboard (#11)
// - share DB กับ: #7 provider-game-api
// - DB_NAME = "lotto_provider" (ต่างจาก standalone = "lotto_standalone")
// - Port: 9081 (game-api ใช้ 9080)
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port string
	Env  string

	// Database — share กับ provider-game-api (#7)
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// JWT — admin กับ operator ใช้ secret แยกกัน
	AdminJWTSecret      string
	AdminJWTExpiryHours int
	OperatorJWTSecret      string
	OperatorJWTExpiryHours int
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "9081"),
		Env:  getEnv("ENV", "development"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "lotto_provider"), // ⭐ DB แยกจาก standalone

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 1),

		AdminJWTSecret:         getEnv("ADMIN_JWT_SECRET", "admin-secret-change-in-production"),
		AdminJWTExpiryHours:    getEnvInt("ADMIN_JWT_EXPIRY_HOURS", 8),
		OperatorJWTSecret:      getEnv("OPERATOR_JWT_SECRET", "operator-secret-change-in-production"),
		OperatorJWTExpiryHours: getEnvInt("OPERATOR_JWT_EXPIRY_HOURS", 24),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" { return val }
	return defaultVal
}
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil { return i }
	}
	return defaultVal
}
