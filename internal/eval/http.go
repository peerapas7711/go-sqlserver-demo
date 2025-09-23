package eval

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Handler struct{ Repo *Repo }

func NewHandler(r *Repo) *Handler { return &Handler{Repo: r} }

func (h *Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/forms", h.createForm)
	r.Get("/forms", h.listForms)

	r.Post("/forms/:id/add/kpis", h.addMyKPIsBulk)
	r.Get("/forms/:id/getdata/kpis", h.listMyKPIs)

	r.Put("/forms/:id/replace/kpis", h.replaceMyKPIs)        // replace-all
	r.Put("/forms/:id/update/kpis/:kpiId", h.updateMyKPI)    // update one (full)
	r.Delete("/forms/:id/detele/kpis/:kpiId", h.deleteMyKPI) // delete one

	r.Post("/forms/:id/competencies", h.addCompsBulk)
	r.Get("/forms/:id/competencies", h.listComps)

	r.Post("/forms/:id/save", h.saveAll)
	r.Get("/forms/:id/getdataform", h.getMyData)

	r.Post("/forms/:id/steps/add", h.addEvalSteps)
	r.Get("/forms/:id/steps", h.listEvalSteps)
	r.Put("/forms/:id/steps/:stepId", h.updateEvalStep)
}

func (h *Handler) createForm(c *fiber.Ctx) error {
	var in CreateFormInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}
	if in.TotalWeight == 0 {
		in.TotalWeight = 100
	} // กัน user ลืม
	f, err := h.Repo.CreateForm(c.Context(), in)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(f)
}

func (h *Handler) listForms(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	fs, err := h.Repo.ListForms(c.Context(), limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": fs})
}

func (h *Handler) addMyKPIsBulk(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))

	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var in MyKPIBulkInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}
	if len(in.Items) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "items required"})
	}

	// สร้าง/ดึง assignment ของ user กับฟอร์มนี้
	a, err := h.Repo.EnsureAssignment(c.Context(), formID, uid, in.DueDate)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	out, err := h.Repo.AddMyKPIsBulk(c.Context(), a.ID, in.Items)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"assignment_id": a.ID, "data": out})
}

func (h *Handler) listMyKPIs(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	out, err := h.Repo.ListMyKPIsByForm(c.Context(), formID, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *Handler) replaceMyKPIs(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var in MyKPIBulkInput
	if err := c.BodyParser(&in); err != nil || len(in.Items) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json or empty items"})
	}
	out, err := h.Repo.ReplaceMyKPIs(c.Context(), formID, uid, in.DueDate, in.Items)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *Handler) updateMyKPI(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	kpiID, _ := strconv.Atoi(c.Params("kpiId"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var in MyKPIInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}
	row, ok2, err := h.Repo.UpdateMyKPI(c.Context(), formID, uid, kpiID, in)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if !ok2 {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(row)
}

func (h *Handler) deleteMyKPI(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	kpiID, _ := strconv.Atoi(c.Params("kpiId"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	okDel, err := h.Repo.DeleteMyKPI(c.Context(), formID, uid, kpiID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if !okDel {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.SendStatus(204)
}

// ดึง user id จาก JWT (ใส่ใน c.Locals("user") โดย middleware)
func currentUserID(c *fiber.Ctx) (int, bool) {
	claims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return 0, false
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, false
	}
	return int(sub), true
}

func (h *Handler) addCompsBulk(c *fiber.Ctx) error {
	fid, _ := strconv.Atoi(c.Params("id"))
	var in CompBulkInput
	if err := c.BodyParser(&in); err != nil || len(in.Items) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json or empty items"})
	}
	out, err := h.Repo.AddCompsBulk(c.Context(), fid, in.Items)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": out})
}

func (h *Handler) listComps(c *fiber.Ctx) error {
	fid, _ := strconv.Atoi(c.Params("id"))
	out, err := h.Repo.ListCompsByForm(c.Context(), fid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *Handler) saveAll(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var in SaveAllInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}

	aid, err := h.Repo.SaveAll(c.Context(), formID, uid, in)
	if err != nil {
		if errors.Is(err, ErrAlreadySubmitted) {
			return c.Status(409).JSON(fiber.Map{
				"error": "already submitted: cannot modify",
				"code":  "ALREADY_SUBMITTED",
			})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	sum, err := h.Repo.ComputeSummary(c.Context(), formID, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"assignment_id": aid,
		"status":        in.Status,
		"summary":       sum,
	})
}

func (h *Handler) getMyData(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	data, err := h.Repo.LoadMyFormData(c.Context(), formID, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

// POST /forms/:id/steps/add
func (h *Handler) addEvalSteps(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	// หา assignment
	a, err := h.Repo.EnsureAssignment(c.Context(), formID, uid, "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	var in struct {
		Evaluators []int `json:"evaluators"`
	}
	if err := c.BodyParser(&in); err != nil || len(in.Evaluators) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json or no evaluators"})
	}

	// Self + คนอื่น
	evaluators := append([]int{uid}, in.Evaluators...)

	steps, err := h.Repo.AddEvalSteps(c.Context(), a.ID, evaluators)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"assignment_id": a.ID, "steps": steps})
}

// GET /forms/:id/steps
func (h *Handler) listEvalSteps(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	steps, err := h.Repo.ListEvalSteps(c.Context(), formID, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	type EvalStepResp struct {
		ID            int     `json:"id"`
		EvaluatorName string  `json:"evaluator_name"`
		Status        int     `json:"status"`
		EvalDate      *string `json:"eval_date"`
	}

	out := make([]EvalStepResp, 0, len(steps))
	for _, s := range steps {
		out = append(out, EvalStepResp{
			ID:            s.ID,
			EvaluatorName: s.EvaluatorName,
			Status:        s.Status,
			EvalDate:      s.EvalDate,
		})
	}

	return c.JSON(fiber.Map{"data": out})
}

// PUT /forms/:id/steps/:stepId
func (h *Handler) updateEvalStep(c *fiber.Ctx) error {
	stepID, _ := strconv.Atoi(c.Params("stepId"))

	var in struct {
		Status   int    `json:"status"`
		EvalDate string `json:"eval_date"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
	}

	step, ok, err := h.Repo.UpdateEvalStep(c.Context(), stepID, in.Status, in.EvalDate)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(step)
}
