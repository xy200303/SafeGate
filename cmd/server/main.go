package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"ip_check/internal/config"
	"ip_check/internal/db"
	"ip_check/internal/handler"
	"ip_check/internal/middleware"
	rediscache "ip_check/internal/redis"
	"ip_check/internal/repository"
	"ip_check/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	gormDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	redisClient := rediscache.New(cfg.RedisAddr, cfg.RedisPassword)
	if err := redisClient.Ping(context.Background()); err != nil {
		log.Printf("redis ping failed: %v", err)
	}
	defer redisClient.Close()

	repo := repository.New(gormDB)

	authService := service.NewAuthService(repo, redisClient, cfg)
	domainService := service.NewDomainService(repo)
	ruleService := service.NewRuleService(repo, redisClient)
	proxyService := service.NewProxyService(repo, redisClient)

	seedPwd, err := authService.SeedAdmin(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if seedPwd != "" {
		cfg.PrintAdminPassword(seedPwd)
	}

	h := handler.New(authService, domainService, ruleService, proxyService)

	adminEngine := gin.Default()
	adminEngine.Use(middleware.CORS(cfg.CORSOrigin))
	h.RegisterAdmin(adminEngine, middleware.Auth(authService))

	if cfg.Mode == "all" {
		if err := serveStatic(adminEngine, cfg.WebDist); err != nil {
			log.Printf("static mount warning: %v", err)
		}
	}

	proxyEngine := gin.Default()
	proxyEngine.NoRoute(h.Proxy())

	adminAddr := net.JoinHostPort(cfg.Host, cfg.AdminPort)
	proxyAddr := net.JoinHostPort(cfg.Host, cfg.Port)

	errCh := make(chan error, 2)
	go func() {
		log.Printf("admin listening on %s", adminAddr)
		errCh <- http.ListenAndServe(adminAddr, adminEngine)
	}()
	go func() {
		log.Printf("proxy listening on %s", proxyAddr)
		errCh <- http.ListenAndServe(proxyAddr, proxyEngine)
	}()

	return <-errCh
}

func serveStatic(r *gin.Engine, dist string) error {
	info, err := os.Stat(dist)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("web dist directory not found: %s", dist)
	}
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Status(http.StatusNotFound)
			return
		}
		reqPath := path.Clean("/" + c.Request.URL.Path)
		filePath := filepath.Join(dist, filepath.FromSlash(reqPath))
		fi, err := os.Stat(filePath)
		if os.IsNotExist(err) || fi.IsDir() {
			c.File(filepath.Join(dist, "index.html"))
			return
		}
		c.File(filePath)
	})
	return nil
}
