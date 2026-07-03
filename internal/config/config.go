package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Host           string
	Port           string
	AdminPort      string
	Mode           string
	WebDist        string
	AdminUsername  string
	AdminPassword  string
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	JWTSecret      string
	JWTExpireHours int
	CORSOrigin     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		jwtSecret = hex.EncodeToString(b)
	}

	expire, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	if expire <= 0 {
		expire = 24
	}

	cfg := &Config{
		Host:           getEnv("HOST", ""),
		Port:           getEnv("PORT", "8080"),
		AdminPort:      getEnv("ADMIN_PORT", "8081"),
		Mode:           strings.ToLower(getEnv("MODE", "api")),
		WebDist:        getEnv("WEB_DIST", "web/dist"),
		AdminUsername:  getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://safegate:safegate@postgres:5432/safegate?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),
		JWTSecret:      jwtSecret,
		JWTExpireHours: expire,
		CORSOrigin:     getEnv("CORS_ORIGIN", "*"),
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func (c *Config) PrintAdminPassword(pwd string) {
	fmt.Printf("\n========================================\n")
	fmt.Printf("Admin API port: %s\n", c.AdminPort)
	fmt.Printf("Username: %s\n", c.AdminUsername)
	fmt.Printf("Password: %s\n", pwd)
	fmt.Printf("========================================\n\n")
}
