package routes

import (
	"database/sql"
	"time"

	"go-sqlserver-demo/internal/auth"
	"go-sqlserver-demo/internal/eval"
	"go-sqlserver-demo/internal/timeattendance"
	"go-sqlserver-demo/internal/user"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Options struct {
	DB        *sql.DB
	JWTSecret string
	JWTIssuer string
	JWTTTL    time.Duration
}

func Register(app *fiber.App, opt Options) {

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Authorization, Content-Type",
	}))

	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"name": "go-sqlserver-demo", "ok": true}) })

	api := app.Group("/api")

	repo := user.NewRepo(opt.DB)
	uh := user.NewHandler(repo)
	uh.RegisterRoutes(api.Group("/users")) // auth.JWTMiddleware(opt.JWTSecret)

	ah := auth.NewAuthHandler(opt.DB, opt.JWTSecret, opt.JWTIssuer, opt.JWTTTL)
	ag := api.Group("/auth")
	ag.Post("/login", ah.Login)
	ag.Get("/user", auth.JWTMiddleware(opt.JWTSecret), ah.GetUser)

	taRepo := timeattendance.NewRepo(opt.DB)
	taH := timeattendance.NewHandler(taRepo)
	taH.RegisterRoutes(api.Group("/users", auth.JWTMiddleware(opt.JWTSecret)))

	evRepo := eval.NewRepo(opt.DB)
	evH := eval.NewHandler(evRepo)
	evH.RegisterRoutes(api.Group("/eval", auth.JWTMiddleware(opt.JWTSecret)))

}
