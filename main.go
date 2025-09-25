package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"go-sqlserver-demo/internal/db"
	"go-sqlserver-demo/internal/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func mustEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	_ = godotenv.Load(".env.local") // ถ้ามีจะ override
	// _ = godotenv.Load() // โหลด .env ปกติ (ใช้กับ Docker)

	app := fiber.New()
	app.Use(logger.New())

	// --- DB ---
	port, _ := strconv.Atoi(mustEnv("DB_PORT", "1433"))
	cfg := db.Config{
		Server:   mustEnv("DB_SERVER", "localhost"),
		Port:     port,
		User:     mustEnv("DB_USER", "sa"),
		Password: mustEnv("DB_PASSWORD", "Pete.181042"),
		Database: mustEnv("DB_NAME", "GoDemoDB"),
	}
	sqlDB, err := db.Open(cfg)
	if err != nil {
		log.Fatal("db open:", err)
	}
	defer sqlDB.Close()

	// --- JWT config ---
	jwtSecret := mustEnv("JWT_SECRET", "change-me-please")
	jwtIssuer := mustEnv("JWT_ISSUER", "go-demo")
	jwtTTLStr := mustEnv("JWT_TTL", "15m")
	jwtTTL, err := time.ParseDuration(jwtTTLStr)
	if err != nil {
		jwtTTL = 15 * time.Minute
	}

	routes.Register(app, routes.Options{
		DB:        sqlDB,
		JWTSecret: jwtSecret,
		JWTIssuer: jwtIssuer,
		JWTTTL:    jwtTTL,
	})

	addr := ":" + mustEnv("APP_PORT", "8080")
	log.Println("listening on", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
