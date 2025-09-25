package timeattendance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Handler struct{ Repo *Repo }

func NewHandler(r *Repo) *Handler { return &Handler{Repo: r} }

// ✅ เส้นทางใช้ token ทั้งคู่
func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/timeattendance/v2", h.get)
	r.Put("/timeattendance", h.put)
	r.Get("/:id/timeattendance", h.getbyID)
	r.Put("/:id/timeattendance", h.putbyID)
}

func userIDFromClaims(c *fiber.Ctx) (int, error) {
	claims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("no jwt claims")
	}
	log.Printf("[TA] claims = %#v", claims)

	for _, k := range []string{"id", "sub"} {
		if v, ok := claims[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t), nil
			case json.Number:
				n, _ := t.Int64()
				return int(n), nil
			case string:
				if n, err := strconv.Atoi(t); err == nil {
					return n, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("no id/sub in claims")
}

func (h *Handler) get(c *fiber.Ctx) error {
	uid, err := userIDFromClaims(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token claims"})
	}
	log.Printf("[TA][GET] uid=%d", uid)

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	out, ok, err := h.Repo.Get(ctx, uid)
	if err != nil {
		log.Printf("[TA][GET] repo error uid=%d: %v", uid, err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if !ok {
		log.Printf("[TA][GET] no data -> return empty payload, uid=%d", uid)
		out = Payload{
			TimeAttendance: []Item{},
			ScoreMax:       0,
			ScoreObtained:  0,
			PenaltyTotal:   0,
		}
	}

	return c.JSON(out)
}

func (h *Handler) put(c *fiber.Ctx) error {
	uid, err := userIDFromClaims(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token claims"})
	}

	var in Payload
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	if err := h.Repo.Upsert(ctx, uid, in); err != nil {
		log.Printf("[TA][PUT] upsert error uid=%d: %v", uid, err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	out, ok, err := h.Repo.Get(ctx, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "upserted but not found"})
	}
	return c.JSON(out)
}

func (h *Handler) getbyID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	out, ok, err := h.Repo.Get(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})

	}

	return c.JSON(out)
}

func (h *Handler) putbyID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var in Payload
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	if err := h.Repo.Upsert(ctx, id, in); err != nil {

		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)

}
