package eval

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-pdf/fpdf"
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

	r.Get("/forms/:id/report.pdf", h.reportPDF)

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

// Report PDF
func (h *Handler) reportPDF(c *fiber.Ctx) error {
	formID, _ := strconv.Atoi(c.Params("id"))
	uid, ok := currentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	// 1) ดึงข้อมูลการประเมินของผู้ใช้ (โค้ดที่คุณมีอยู่แล้ว)
	data, err := h.Repo.LoadMyFormData(c.Context(), formID, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 2) สร้าง PDF (ใช้ go-pdf/fpdf แบบเบา ไม่ต้องมี Chrome)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// ใส่ฟอนต์ไทย (แนะนำวางไฟล์ฟอนต์ไว้ข้าง binary เช่น NotoSansThai-Regular.ttf)
	// ถ้าไม่มีฟอนต์ไทย ให้คอมเมนต์สองบรรทัดนี้ออกแล้วใช้ Arial ชั่วคราว
	pdf.AddUTF8Font("Noto", "", "fonts/NotoSansThai-Regular.ttf")
	if err := pdf.Error(); err != nil {
		// fallback เป็น Arial ชั่วคราว
		pdf.SetFont("Arial", "", 14)
	} else {
		pdf.SetFont("Noto", "", 14)
	}

	pdf.Cell(0, 8, fmt.Sprintf("รายงานการประเมินตนเอง (Form %d)", formID))
	pdf.Ln(10)
	pdf.SetFont("Noto", "", 12)
	pdf.Cell(0, 6, fmt.Sprintf("สรุปรวม: %.2f%%   เกรด: %s", data.Summary.TotalPct, data.Summary.Grade))
	pdf.Ln(8)

	// --- ตาราง KPI ---
	pdf.SetFont("Noto", "", 12)
	pdf.Cell(0, 7, "KPI")
	pdf.Ln(7)

	// header
	w := []float64{10, 75, 20, 20, 20} // #, ชื่อ, Max, Weight, Score
	pdf.SetFillColor(245, 245, 245)
	for i, h := range []string{"ลำดับ", "รายการ", "Max", "Weight", "Score"} {
		align := "L"
		if i > 1 {
			align = "R"
		}
		pdf.CellFormat(w[i], 7, h, "1", 0, align, true, 0, "")
	}
	pdf.Ln(-1)

	// rows
	pdf.SetFillColor(255, 255, 255)
	for i, k := range data.KPIs {
		pdf.CellFormat(w[0], 7, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(w[1], 7, fmt.Sprintf("[%s] %s", k.Code, k.Title), "1", 0, "L", false, 0, "")
		pdf.CellFormat(w[2], 7, fmt.Sprintf("%.1f", k.MaxScore), "1", 0, "R", false, 0, "")
		pdf.CellFormat(w[3], 7, fmt.Sprintf("%d", k.Weight), "1", 0, "R", false, 0, "")
		pdf.CellFormat(w[4], 7, fmt.Sprintf("%.1f", k.Score), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	// --- ตาราง Competency ---
	pdf.Ln(3)
	pdf.Cell(0, 7, "Competency")
	pdf.Ln(7)

	wc := []float64{10, 85, 25, 25, 25} // #, ชื่อ, Max, Weight, Score
	pdf.SetFillColor(245, 245, 245)
	for i, h := range []string{"ลำดับ", "หัวข้อ", "Max", "Weight", "Score"} {
		align := "L"
		if i > 1 {
			align = "R"
		}
		pdf.CellFormat(wc[i], 7, h, "1", 0, align, true, 0, "")
	}
	pdf.Ln(-1)
	for i, cpt := range data.Competencies {
		pdf.CellFormat(wc[0], 7, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(wc[1], 7, cpt.Title, "1", 0, "L", false, 0, "")
		pdf.CellFormat(wc[2], 7, fmt.Sprintf("%.1f", cpt.MaxScore), "1", 0, "R", false, 0, "")
		pdf.CellFormat(wc[3], 7, fmt.Sprintf("%.2f", cpt.Weight), "1", 0, "R", false, 0, "")
		pdf.CellFormat(wc[4], 7, fmt.Sprintf("%.1f", cpt.Score), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	// --- TA / Notes ---
	pdf.Ln(4)
	pdf.Cell(0, 6, fmt.Sprintf("Time Attendance: %.1f / %.1f (%.1f%%)",
		data.TimeAttendance.Score, data.TimeAttendance.FullScore,
		(data.TimeAttendance.Score/data.TimeAttendance.FullScore)*100))
	pdf.Ln(8)

	// 3) ส่งเป็นไฟล์ดาวน์โหลดทันที
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=eval-report.pdf")
	return c.SendStream(bytes.NewReader(buf.Bytes()))
}
