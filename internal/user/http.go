package user

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	Repo *Repo
}

func NewHandler(r *Repo) *Handler { return &Handler{Repo: r} }

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Get("/", h.list)
	r.Get("/:id", h.get)
	r.Post("/", h.create)
	r.Patch("/:id", h.update)
	r.Delete("/:id", h.remove)
}

func (h *Handler) list(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	ctx := c.Context()
	users, err := h.Repo.List(ctx, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data": users,
	})
}

func (h *Handler) get(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	u, err := h.Repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(u)
}

func (h *Handler) create(c *fiber.Ctx) error {
	var in CreateUserInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}
	u, err := h.Repo.Create(c.Context(), in)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(u)
}

func (h *Handler) update(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var in UpdateUserInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}
	u, err := h.Repo.Update(c.Context(), id, in)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(u)
}

func (h *Handler) remove(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := h.Repo.Delete(c.Context(), id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}
