package timeattendance

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ Repo *Repo }

func NewHandler(r *Repo) *Handler { return &Handler{Repo: r} }

func (h *Handler) RegisterRoutes(r fiber.Router) {
	// /api/users/:id/time-attendance
	r.Get("/:id/timeattendance", h.get)
	r.Put("/:id/timeattendance", h.put)
}

func (h *Handler) get(c *fiber.Ctx) error {
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

func (h *Handler) put(c *fiber.Ctx) error {
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
