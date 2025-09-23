package eval

type Form struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	TitleTH       string `json:"title_th"`
	TitleEN       string `json:"title_en,omitempty"`
	KPIWeight     int    `json:"kpi_weight"`
	CompWeight    int    `json:"comp_weight"`
	TAWeight      int    `json:"ta_weight"`
	TotalWeight   int    `json:"total_weight"`
	CalcMethod    int    `json:"calc_method"`  // การคำนวณคะแนน
	ScoreScheme   int    `json:"score_scheme"` // รูปแบบการให้คะแนน
	KPICfgStart   string `json:"kpi_cfg_start,omitempty"`
	KPICfgEnd     string `json:"kpi_cfg_end,omitempty"`
	EvalStart     string `json:"eval_start,omitempty"`
	EvalEnd       string `json:"eval_end,omitempty"`
	OtherLvStart  string `json:"other_lv_start,omitempty"`
	OtherLvEnd    string `json:"other_lv_end,omitempty"`
	AnnualLvStart string `json:"annual_lv_start,omitempty"`
	AnnualLvEnd   string `json:"annual_lv_end,omitempty"`
	Remark        string `json:"remark,omitempty"`
}

type CreateFormInput struct {
	Code          string `json:"code"`
	TitleTH       string `json:"title_th"`
	TitleEN       string `json:"title_en"`
	KPIWeight     int    `json:"kpi_weight"`
	CompWeight    int    `json:"comp_weight"`
	TAWeight      int    `json:"ta_weight"`
	TotalWeight   int    `json:"total_weight"` // ปกติ 100
	CalcMethod    int    `json:"calc_method"`
	ScoreScheme   int    `json:"score_scheme"`
	KPICfgStart   string `json:"kpi_cfg_start"`
	KPICfgEnd     string `json:"kpi_cfg_end"`
	EvalStart     string `json:"eval_start"`
	EvalEnd       string `json:"eval_end"`
	OtherLvStart  string `json:"other_lv_start"`
	OtherLvEnd    string `json:"other_lv_end"`
	AnnualLvStart string `json:"annual_lv_start"`
	AnnualLvEnd   string `json:"annual_lv_end"`
	Remark        string `json:"remark"`
}

// การมอบหมายฟอร์มให้ผู้ใช้
type Assign struct {
	ID      int    `json:"id"`
	FormID  int    `json:"form_id"`
	UserID  int    `json:"user_id"`
	Status  int    `json:"status"`
	DueDate string `json:"due_date,omitempty"`
}

// KPI ของผู้ใช้ (ต่อ assignment)
type MyKPIItem struct {
	ID            int     `json:"id"`
	AssignmentID  int     `json:"assignment_id"`
	Idx           int     `json:"idx"`       // ลำดับ
	Code          string  `json:"code"`      // รหัสหัวข้อการประเมิน
	Title         string  `json:"title"`     // หัวข้อการประเมิน
	MaxScore      float64 `json:"max_score"` // คะแนนเต็ม
	Weight        int     `json:"weight"`    // ค่าถ่วงน้ำหนัก (%)
	ExpectedScore float64 `json:"expected_score"`
	Score         float64 `json:"score"`    // คะแนน
	Note          string  `json:"note"`     // หมายเหตุ
	Measure       string  `json:"measure"`  // วิธีการวัด
	Criteria      string  `json:"criteria"` // เกณฑ์การให้คะแนน
	Unit          string  `json:"unit,omitempty"`
}

type MyKPIInput struct {
	Idx           int     `json:"idx"`
	Code          string  `json:"code"`
	Title         string  `json:"title"`
	MaxScore      float64 `json:"max_score"`
	Weight        int     `json:"weight"`
	ExpectedScore float64 `json:"expected_score"`
	Score         float64 `json:"score"`
	Note          string  `json:"note"`
	Measure       string  `json:"measure"`
	Criteria      string  `json:"criteria"`
	Unit          string  `json:"unit"`
}

type MyKPIBulkInput struct {
	Items   []MyKPIInput `json:"items"`
	DueDate string       `json:"due_date,omitempty"` // เผื่อตั้งกำหนดส่งตอนสร้าง assignment
}

// ==== Competency items (shared by form) ====

type CompItem struct {
	ID            int     `json:"id"`
	FormID        int     `json:"form_id"`
	Idx           int     `json:"idx"`
	Title         string  `json:"title"`
	MaxScore      float64 `json:"max_score"`
	Weight        float64 `json:"weight"`         // ค่าถ่วงน้ำหนัก (0.70 หรือ 70 ตามที่ UI ส่ง)
	FullTotal     float64 `json:"full_total"`     // คะแนนเต็มรวม
	ExpectedScore float64 `json:"expected_score"` // คะแนนคาดหวัง
}

type CompInput struct {
	Idx           int     `json:"idx"`
	Title         string  `json:"title"`
	MaxScore      float64 `json:"max_score"`
	Weight        float64 `json:"weight"`
	FullTotal     float64 `json:"full_total"`
	ExpectedScore float64 `json:"expected_score"`
}

type CompBulkInput struct {
	Items []CompInput `json:"items"`
}

// Save
type SaveStatus string

const (
	SaveDraft  SaveStatus = "draft"
	SaveSubmit SaveStatus = "submitted"
)

type SaveKPIInput = MyKPIInput // ใช้โครงเดิมที่คุณมี (idx, code, title, max_score, weight, expected_score, score, note, measure, criteria, unit)

type CompScoreInput struct {
	CompID int     `json:"comp_id"`
	Score  float64 `json:"score"`
	Note   string  `json:"note"`
}

type TAScoreInput struct {
	FullScore float64 `json:"full_score"`
	Score     float64 `json:"score"`
}

type DevPlanItemInput struct {
	Idx      int    `json:"idx"`
	Content  string `json:"content"`
	Priority string `json:"priority"` // High/Medium/Low
	Timing   string `json:"timing"`   // yyyy-mm-dd
	Remarks  string `json:"remarks"`
}

type AdditionalInput struct {
	Q1 string `json:"q1"`
	Q2 string `json:"q2"`
	Q3 string `json:"q3"`
	Q4 string `json:"q4"`
	Q5 string `json:"q5"`
}

type SaveAllInput struct {
	Status           SaveStatus         `json:"status"`
	DueDate          string             `json:"due_date,omitempty"`
	KPIs             []SaveKPIInput     `json:"kpis"`
	CompetencyScores []CompScoreInput   `json:"competency_scores"`
	TimeAttendance   TAScoreInput       `json:"time_attendance"`
	DevelopmentPlan  []DevPlanItemInput `json:"development_plan"`
	Additional       AdditionalInput    `json:"additional"`
}

type EvalSummary struct {
	FormID     int     `json:"form_id"`
	KPIWeight  int     `json:"kpi_weight"`
	KPIPct     float64 `json:"kpi_pct"`
	CompWeight int     `json:"comp_weight"`
	CompPct    float64 `json:"comp_pct"`
	TAWeight   int     `json:"ta_weight"`
	TAPct      float64 `json:"ta_pct"`
	TotalPct   float64 `json:"total_pct"`
	Grade      string  `json:"grade"`     // "N/A" ถ้าไม่มี band
	GradeMin   float64 `json:"grade_min"` // ช่วงคะแนนของเกรด (ถ้ามี)
	GradeMax   float64 `json:"grade_max"`
}

// คะแนน competency พร้อมหัวข้อ
type MyCompWithScore struct {
	CompID        int     `json:"comp_id"`
	Idx           int     `json:"idx"`
	Title         string  `json:"title"`
	MaxScore      float64 `json:"max_score"`
	Weight        float64 `json:"weight"`
	FullTotal     float64 `json:"full_total"`
	ExpectedScore float64 `json:"expected_score"`
	Score         float64 `json:"score"`
	Note          string  `json:"note"`
}

// response สำหรับโหลดทุกแท็บ
type LoadMyFormData struct {
	AssignmentID    int                `json:"assignment_id"`
	Status          int                `json:"status"`
	DueDate         string             `json:"due_date,omitempty"`
	KPIs            []MyKPIItem        `json:"kpis"`
	Competencies    []MyCompWithScore  `json:"competencies"`
	TimeAttendance  TAScoreInput       `json:"time_attendance"`
	DevelopmentPlan []DevPlanItemInput `json:"development_plan"`
	Additional      AdditionalInput    `json:"additional"`
	Summary         EvalSummary        `json:"summary"`
}


type EvalStep struct {
    ID            int    `json:"id"`
    AssignmentID  int    `json:"assignment_id"`
    Idx           int    `json:"idx"`
    EvaluatorID   int    `json:"evaluator_id"`
    EvaluatorName string `json:"evaluator_name"`
    Status        int    `json:"status"`
    EvalDate      *string `json:"eval_date"`
}
