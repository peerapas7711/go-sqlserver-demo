package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go-sqlserver-demo/internal/user"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	Role        string         `json:"role"`
	Company     []user.Company `json:"company"`
	PersonCode  string         `json:"personcode"`
	Position    string         `json:"position"`
	Department  string         `json:"department"`
	UrlImage    string         `json:"urlimage"`
	StartDate   string         `json:"start_date"`
	ConfirmDate string         `json:"confirm_date"`
	YearsOfWork int            `json:"years_of_work"`
}

type LoginResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
	JWTIssuer string
	JWTTtl    time.Duration
}

func NewAuthHandler(db *sql.DB, secret, issuer string, ttl time.Duration) *AuthHandler {
	return &AuthHandler{DB: db, JWTSecret: secret, JWTIssuer: issuer, JWTTtl: ttl}
}

func (h *AuthHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/login", h.Login)
	r.Post("/refreshtoken", h.Refresh)
	r.Get("/user", h.GetUser)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {

		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "personcode and password are required"})
	}

	ctx := context.Background()

	var (
		u                user.User
		pwHash           []byte
		startD, confirmD sql.NullTime
	)
	err := h.DB.QueryRowContext(ctx, `
    SELECT id, name, email, role, password_hash,
           person_code, position, department, url_image,
           TRY_CONVERT(date, NULLIF(start_date, ''))   AS start_date,
           TRY_CONVERT(date, NULLIF(confirm_date, '')) AS confirm_date
    FROM dbo.users
    WHERE person_code = @p1
`, req.Username).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role, &pwHash,
		&u.PersonCode, &u.Position, &u.Department, &u.UrlImage,
		&startD, &confirmD,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if err := bcrypt.CompareHashAndPassword(pwHash, []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	rows, err := h.DB.QueryContext(ctx, `
		SELECT c.id, c.code, c.name, c.image
		FROM dbo.companies c
		INNER JOIN dbo.user_companies uc ON uc.company_id = c.id
		WHERE uc.user_id = @p1
	`, u.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var companies []user.Company
	for rows.Next() {
		var cp user.Company
		if err := rows.Scan(&cp.ID, &cp.Code, &cp.Name, &cp.Image); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		companies = append(companies, cp)
	}
	u.Company = companies

	if startD.Valid {
		u.StartDate = startD.Time.Format("2006-01-02")
	}
	if confirmD.Valid {
		u.ConfirmDate = confirmD.Time.Format("2006-01-02")
	}

	u.YearsOfWork = calcYearsOfWork(confirmD, startD)

	claims := jwt.MapClaims{
		"sub":   u.ID,
		"pc":    u.PersonCode,
		"name":  u.Name,
		"email": u.Email,
		"role":  u.Role,
		"iss":   h.JWTIssuer,
		"exp":   time.Now().Add(h.JWTTtl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "could not sign token"})
	}

	refreshClaims := jwt.MapClaims{
		"sub": u.ID,
		"iss": h.JWTIssuer,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"typ": "refresh",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefresh, err := refreshToken.SignedString([]byte(h.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "could not sign refresh token"})
	}

	respUser := UserResponse{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Role:        u.Role,
		Company:     u.Company,
		PersonCode:  u.PersonCode,
		Position:    u.Position,
		Department:  u.Department,
		UrlImage:    u.UrlImage,
		StartDate:   u.StartDate,
		ConfirmDate: u.ConfirmDate,
		YearsOfWork: u.YearsOfWork,
	}

	return c.JSON(LoginResponse{
		Token:        signed,
		RefreshToken: signedRefresh,
		User:         respUser,
	})
}

func calcYearsOfWork(confirm sql.NullTime, start sql.NullTime) int {
	var base time.Time
	switch {
	case confirm.Valid:
		base = confirm.Time
	case start.Valid:
		base = start.Time
	default:
		return 0
	}
	now := time.Now()
	years := now.Year() - base.Year()
	anniv := time.Date(now.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
	if now.Before(anniv) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}

	token, err := jwt.Parse(body.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "invalid refresh token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "refresh" {
		return c.Status(401).JSON(fiber.Map{"error": "invalid refresh token type"})
	}

	newClaims := jwt.MapClaims{
		"sub": claims["sub"],
		"iss": h.JWTIssuer,
		"exp": time.Now().Add(h.JWTTtl).Unix(),
	}
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	signedNew, _ := newToken.SignedString([]byte(h.JWTSecret))

	return c.JSON(fiber.Map{"token": signedNew})
}

// GET /api/auth/user
func (h *AuthHandler) GetUser(c *fiber.Ctx) error {
	cu := c.Locals("user")
	if cu == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	claims, ok := cu.(jwt.MapClaims)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid claims"})
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid sub"})
	}
	userID := int(sub)

	repo := user.NewRepo(h.DB)
	u, err := repo.GetByID(c.Context(), userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	resp := UserResponse{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Role:        u.Role,
		Company:     u.Company,
		PersonCode:  u.PersonCode,
		Position:    u.Position,
		Department:  u.Department,
		UrlImage:    u.UrlImage,
		StartDate:   u.StartDate,
		ConfirmDate: u.ConfirmDate,
		YearsOfWork: u.YearsOfWork,
	}

	return c.JSON(resp)
}
