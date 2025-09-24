package eval

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Repo struct{ DB *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

func (r *Repo) CreateForm(ctx context.Context, in CreateFormInput) (Form, error) {
	const q = `
INSERT INTO dbo.eval_form
(code,title_th,title_en,kpi_weight,comp_weight,ta_weight,total_weight,calc_method,score_scheme,
 kpi_cfg_start,kpi_cfg_end,eval_start,eval_end,other_lv_start,other_lv_end,annual_lv_start,annual_lv_end,remark)
OUTPUT inserted.id, inserted.code, inserted.title_th, inserted.title_en, inserted.kpi_weight, inserted.comp_weight,
       inserted.ta_weight, inserted.total_weight, inserted.calc_method, inserted.score_scheme,
       CONVERT(varchar(10),inserted.kpi_cfg_start,23),
       CONVERT(varchar(10),inserted.kpi_cfg_end,23),
       CONVERT(varchar(10),inserted.eval_start,23),
       CONVERT(varchar(10),inserted.eval_end,23),
       CONVERT(varchar(10),inserted.other_lv_start,23),
       CONVERT(varchar(10),inserted.other_lv_end,23),
       CONVERT(varchar(10),inserted.annual_lv_start,23),
       CONVERT(varchar(10),inserted.annual_lv_end,23),
       inserted.remark
VALUES
(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8,@p9,
 NULLIF(@p10,''),NULLIF(@p11,''),NULLIF(@p12,''),NULLIF(@p13,''),
 NULLIF(@p14,''),NULLIF(@p15,''),NULLIF(@p16,''),NULLIF(@p17,''), NULLIF(@p18,''));
`
	var f Form
	err := r.DB.QueryRowContext(ctx, q,
		in.Code, in.TitleTH, in.TitleEN, in.KPIWeight, in.CompWeight, in.TAWeight, in.TotalWeight, in.CalcMethod, in.ScoreScheme,
		in.KPICfgStart, in.KPICfgEnd, in.EvalStart, in.EvalEnd, in.OtherLvStart, in.OtherLvEnd, in.AnnualLvStart, in.AnnualLvEnd, in.Remark,
	).Scan(
		&f.ID, &f.Code, &f.TitleTH, &f.TitleEN, &f.KPIWeight, &f.CompWeight, &f.TAWeight, &f.TotalWeight, &f.CalcMethod, &f.ScoreScheme,
		&f.KPICfgStart, &f.KPICfgEnd, &f.EvalStart, &f.EvalEnd, &f.OtherLvStart, &f.OtherLvEnd, &f.AnnualLvStart, &f.AnnualLvEnd, &f.Remark,
	)
	return f, err
}

func (r *Repo) ListForms(ctx context.Context, limit, offset int) ([]Form, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	const q = `
SELECT id, code, title_th, title_en, kpi_weight, comp_weight, ta_weight, total_weight, calc_method, score_scheme,
       CONVERT(varchar(10),kpi_cfg_start,23),
       CONVERT(varchar(10),kpi_cfg_end,23),
       CONVERT(varchar(10),eval_start,23),
       CONVERT(varchar(10),eval_end,23),
       CONVERT(varchar(10),other_lv_start,23),
       CONVERT(varchar(10),other_lv_end,23),
       CONVERT(varchar(10),annual_lv_start,23),
       CONVERT(varchar(10),annual_lv_end,23),
       remark
FROM dbo.eval_form
ORDER BY id DESC
OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY;`
	rows, err := r.DB.QueryContext(ctx, q, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Form
	for rows.Next() {
		var f Form
		if err := rows.Scan(
			&f.ID, &f.Code, &f.TitleTH, &f.TitleEN, &f.KPIWeight, &f.CompWeight, &f.TAWeight, &f.TotalWeight, &f.CalcMethod, &f.ScoreScheme,
			&f.KPICfgStart, &f.KPICfgEnd, &f.EvalStart, &f.EvalEnd, &f.OtherLvStart, &f.OtherLvEnd, &f.AnnualLvStart, &f.AnnualLvEnd, &f.Remark,
		); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// สร้าง (หรือดึง) assignment ของ user กับฟอร์ม
func (r *Repo) EnsureAssignment(ctx context.Context, formID, userID int, due string) (Assign, error) {
	var a Assign
	err := r.DB.QueryRowContext(ctx, `
SELECT id, form_id, user_id, status, CONVERT(varchar(10), due_date, 23)
FROM dbo.eval_assignment
WHERE form_id=@p1 AND user_id=@p2;`, formID, userID).
		Scan(&a.ID, &a.FormID, &a.UserID, &a.Status, &a.DueDate)

	if err == nil {
		// มีอยู่แล้ว → อัปเดต due_date เฉพาะกรณีมีส่งมาใหม่
		if strings.TrimSpace(due) != "" {
			_, _ = r.DB.ExecContext(ctx, `
UPDATE dbo.eval_assignment
SET due_date = NULLIF(@p1,''), updated_at = SYSUTCDATETIME()
WHERE id = @p2;`, due, a.ID)
			a.DueDate = due
		}
		return a, nil
	}
	if err != sql.ErrNoRows {
		return Assign{}, err
	}

	// ยังไม่มี → กำหนดวันนี้ (Asia/Bangkok) ถ้า FE ไม่ส่ง due_date มา
	if strings.TrimSpace(due) == "" {
		loc, _ := time.LoadLocation("Asia/Bangkok")
		due = time.Now().In(loc).Format("2006-01-02")
	}

	const ins = `
INSERT INTO dbo.eval_assignment(form_id, user_id, status, due_date)
OUTPUT inserted.id, inserted.form_id, inserted.user_id, inserted.status,
       CONVERT(varchar(10), inserted.due_date, 23)
VALUES(@p1, @p2, 0, @p3);`
	err = r.DB.QueryRowContext(ctx, ins, formID, userID, due).
		Scan(&a.ID, &a.FormID, &a.UserID, &a.Status, &a.DueDate)
	return a, err
}

// เพิ่ม KPI ให้ assignment (bulk append)
func (r *Repo) AddMyKPIsBulk(ctx context.Context, assignmentID int, in []MyKPIInput) ([]MyKPIItem, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_kpi_user
(assignment_id, idx, code, title, max_score, weight, expected_score, score, note, measure, criteria, unit)
OUTPUT inserted.id, inserted.assignment_id, inserted.idx,
       ISNULL(inserted.code,''), inserted.title, inserted.max_score, inserted.weight,
       inserted.expected_score, inserted.score, ISNULL(inserted.note,''),
       ISNULL(inserted.measure,''), ISNULL(inserted.criteria,''), ISNULL(inserted.unit,'')
VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8, NULLIF(@p9,''), NULLIF(@p10,''), NULLIF(@p11,''), NULLIF(@p12,''));`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var out []MyKPIItem
	for _, it := range in {
		var row MyKPIItem
		if err = stmt.QueryRowContext(ctx,
			assignmentID, it.Idx, it.Code, it.Title, it.MaxScore, it.Weight, it.ExpectedScore, it.Score,
			it.Note, it.Measure, it.Criteria, it.Unit,
		).Scan(
			&row.ID, &row.AssignmentID, &row.Idx,
			&row.Code, &row.Title, &row.MaxScore, &row.Weight,
			&row.ExpectedScore, &row.Score, &row.Note,
			&row.Measure, &row.Criteria, &row.Unit,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}

	_, err = tx.ExecContext(ctx, `UPDATE dbo.eval_kpi_user SET updated_at=SYSUTCDATETIME() WHERE assignment_id=@p1;`, assignmentID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	return out, err
}

// ดึง KPI ของ user สำหรับฟอร์ม
func (r *Repo) ListMyKPIsByForm(ctx context.Context, formID, userID int) ([]MyKPIItem, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT ku.id, ku.assignment_id, ku.idx,
       ISNULL(ku.code,''), ku.title, ku.max_score, ku.weight,
       ku.expected_score, ku.score, ISNULL(ku.note,''),
       ISNULL(ku.measure,''), ISNULL(ku.criteria,''), ISNULL(ku.unit,'')
FROM dbo.eval_kpi_user ku
JOIN dbo.eval_assignment a ON a.id = ku.assignment_id
WHERE a.form_id=@p1 AND a.user_id=@p2
ORDER BY ku.idx, ku.id;`, formID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MyKPIItem
	for rows.Next() {
		var k MyKPIItem
		if err := rows.Scan(
			&k.ID, &k.AssignmentID, &k.Idx,
			&k.Code, &k.Title, &k.MaxScore, &k.Weight,
			&k.ExpectedScore, &k.Score, &k.Note,
			&k.Measure, &k.Criteria, &k.Unit,
		); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// Replace KPI ทั้งชุดของ "ผู้ใช้" ภายใต้ "ฟอร์ม" (ลบของเก่าทั้งหมดก่อนแล้วค่อย insert ใหม่)
func (r *Repo) ReplaceMyKPIs(ctx context.Context, formID, userID int, due string, in []MyKPIInput) ([]MyKPIItem, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// ensure assignment (สร้างถ้ายังไม่มี / อัปเดต due ถ้ามี)
	a, err := r.EnsureAssignment(ctx, formID, userID, due)
	if err != nil {
		return nil, err
	}

	// ลบชุดเดิมทั้งหมด
	if _, err = tx.ExecContext(ctx, `DELETE FROM dbo.eval_kpi_user WHERE assignment_id=@p1;`, a.ID); err != nil {
		return nil, err
	}

	// เตรียม insert ใหม่ทั้งหมด
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_kpi_user
(assignment_id, idx, code, title, max_score, weight, expected_score, score, note, measure, criteria, unit)
OUTPUT inserted.id, inserted.assignment_id, inserted.idx,
       ISNULL(inserted.code,''), inserted.title, inserted.max_score, inserted.weight,
       inserted.expected_score, inserted.score, ISNULL(inserted.note,''),
       ISNULL(inserted.measure,''), ISNULL(inserted.criteria,''), ISNULL(inserted.unit,'')
VALUES(@p1,@p2, NULLIF(@p3,''), @p4, @p5, @p6, @p7, @p8, NULLIF(@p9,''), NULLIF(@p10,''), NULLIF(@p11,''), NULLIF(@p12,''));`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var out []MyKPIItem
	for _, it := range in {
		var row MyKPIItem
		if err = stmt.QueryRowContext(ctx,
			a.ID, it.Idx, it.Code, it.Title, it.MaxScore, it.Weight, it.ExpectedScore, it.Score,
			it.Note, it.Measure, it.Criteria, it.Unit,
		).Scan(
			&row.ID, &row.AssignmentID, &row.Idx,
			&row.Code, &row.Title, &row.MaxScore, &row.Weight,
			&row.ExpectedScore, &row.Score, &row.Note,
			&row.Measure, &row.Criteria, &row.Unit,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}

	_, err = tx.ExecContext(ctx, `UPDATE dbo.eval_kpi_user SET updated_at=SYSUTCDATETIME() WHERE assignment_id=@p1;`, a.ID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	return out, err
}

// Update KPI "รายการเดียว" ของผู้ใช้ในฟอร์ม (ยืนยันสิทธิ์ด้วย form_id+user_id)
func (r *Repo) UpdateMyKPI(ctx context.Context, formID, userID, kpiID int, in MyKPIInput) (MyKPIItem, bool, error) {
	const q = `
UPDATE ku
SET ku.idx=@p4, ku.code=NULLIF(@p5,''), ku.title=@p6, ku.max_score=@p7, ku.weight=@p8,
    ku.expected_score=@p9, ku.score=@p10, ku.note=NULLIF(@p11,''),
    ku.measure=NULLIF(@p12,''), ku.criteria=NULLIF(@p13,''), ku.unit=NULLIF(@p14,''),
    ku.updated_at=SYSUTCDATETIME()
OUTPUT inserted.id, inserted.assignment_id, inserted.idx,
       ISNULL(inserted.code,''), inserted.title, inserted.max_score, inserted.weight,
       inserted.expected_score, inserted.score, ISNULL(inserted.note,''),
       ISNULL(inserted.measure,''), ISNULL(inserted.criteria,''), ISNULL(inserted.unit,'')
FROM dbo.eval_kpi_user ku
JOIN dbo.eval_assignment a ON a.id = ku.assignment_id
WHERE ku.id=@p1 AND a.form_id=@p2 AND a.user_id=@p3;`

	var row MyKPIItem
	err := r.DB.QueryRowContext(ctx, q,
		kpiID, formID, userID,
		in.Idx, in.Code, in.Title, in.MaxScore, in.Weight,
		in.ExpectedScore, in.Score, in.Note, in.Measure, in.Criteria, in.Unit,
	).Scan(
		&row.ID, &row.AssignmentID, &row.Idx,
		&row.Code, &row.Title, &row.MaxScore, &row.Weight,
		&row.ExpectedScore, &row.Score, &row.Note,
		&row.Measure, &row.Criteria, &row.Unit,
	)
	if err == sql.ErrNoRows {
		return MyKPIItem{}, false, nil
	}
	if err != nil {
		return MyKPIItem{}, false, err
	}
	return row, true, nil
}

// Delete KPI "รายการเดียว" ของผู้ใช้ในฟอร์ม (ยืนยันสิทธิ์ด้วย form_id+user_id)
func (r *Repo) DeleteMyKPI(ctx context.Context, formID, userID, kpiID int) (bool, error) {
	res, err := r.DB.ExecContext(ctx, `
DELETE ku
FROM dbo.eval_kpi_user ku
JOIN dbo.eval_assignment a ON a.id = ku.assignment_id
WHERE ku.id=@p1 AND a.form_id=@p2 AND a.user_id=@p3;`, kpiID, formID, userID)
	if err != nil {
		return false, err
	}
	aff, _ := res.RowsAffected()
	return aff > 0, nil
}

// เพิ่ม Competency หลายข้อ (append) ให้ฟอร์ม
func (r *Repo) AddCompsBulk(ctx context.Context, formID int, in []CompInput) ([]CompItem, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_competency(form_id, idx, title, max_score, weight, full_total, expected_score)
OUTPUT inserted.id, inserted.form_id, inserted.idx, inserted.title,
       inserted.max_score, inserted.weight, inserted.full_total, inserted.expected_score
VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7);`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var out []CompItem
	for _, it := range in {
		var row CompItem
		if err = stmt.QueryRowContext(ctx,
			formID, it.Idx, it.Title, it.MaxScore, it.Weight, it.FullTotal, it.ExpectedScore,
		).Scan(&row.ID, &row.FormID, &row.Idx, &row.Title, &row.MaxScore, &row.Weight, &row.FullTotal, &row.ExpectedScore); err != nil {
			return nil, err
		}
		out = append(out, row)
	}

	_, err = tx.ExecContext(ctx, `UPDATE dbo.eval_competency SET updated_at=SYSUTCDATETIME() WHERE form_id=@p1;`, formID)
	if err != nil {
		return nil, err
	}

	return out, tx.Commit()
}

// ดึง Competency ของฟอร์ม
func (r *Repo) ListCompsByForm(ctx context.Context, formID int) ([]CompItem, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, form_id, idx, title, max_score, weight, full_total, expected_score
FROM dbo.eval_competency
WHERE form_id=@p1
ORDER BY idx, id;`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompItem
	for rows.Next() {
		var c CompItem
		if err := rows.Scan(&c.ID, &c.FormID, &c.Idx, &c.Title, &c.MaxScore, &c.Weight, &c.FullTotal, &c.ExpectedScore); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var ErrAlreadySubmitted = errors.New("already submitted: cannot modify")

func (r *Repo) SaveAll(ctx context.Context, formID, userID int, in SaveAllInput) (int, error) {
	// 1) assignment
	a, err := r.EnsureAssignment(ctx, formID, userID, in.DueDate)
	if err != nil {
		return 0, err
	}

	if a.Status == 1 {
		return 0, ErrAlreadySubmitted
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	/* ---- KPI (replace ทั้งชุด) ---- */
	if _, err = tx.ExecContext(ctx, `DELETE FROM dbo.eval_kpi_user WHERE assignment_id=@p1;`, a.ID); err != nil {
		return 0, err
	}
	if len(in.KPIs) > 0 {
		stmtKPI, err2 := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_kpi_user
(assignment_id, idx, code, title, max_score, weight, expected_score, score, note, measure, criteria, unit)
VALUES(@p1,@p2, NULLIF(@p3,''), @p4, @p5, @p6, @p7, @p8, NULLIF(@p9,''), NULLIF(@p10,''), NULLIF(@p11,''), NULLIF(@p12,''));`)
		if err2 != nil {
			return 0, err2
		}
		defer stmtKPI.Close()
		for _, it := range in.KPIs {
			if _, err = stmtKPI.ExecContext(ctx, a.ID, it.Idx, it.Code, it.Title, it.MaxScore, it.Weight, it.ExpectedScore, it.Score, it.Note, it.Measure, it.Criteria, it.Unit); err != nil {
				return 0, err
			}
		}
	}

	/* ---- Competency scores (delete+insert) ---- */
	if _, err = tx.ExecContext(ctx, `DELETE cs FROM dbo.eval_competency_score cs WHERE cs.assignment_id=@p1;`, a.ID); err != nil {
		return 0, err
	}
	if len(in.CompetencyScores) > 0 {
		stmtC, err2 := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_competency_score(assignment_id, comp_id, score, note)
VALUES(@p1,@p2,@p3, NULLIF(@p4,''));`)
		if err2 != nil {
			return 0, err2
		}
		defer stmtC.Close()
		for _, c := range in.CompetencyScores {
			if _, err = stmtC.ExecContext(ctx, a.ID, c.CompID, c.Score, c.Note); err != nil {
				return 0, err
			}
		}
	}

	/* ---- Time Attendance score (upsert) ---- */
	_, err = tx.ExecContext(ctx, `
MERGE dbo.eval_ta_score AS t
USING (SELECT @p1 AS assignment_id, @p2 AS full_score, @p3 AS score) s
ON (t.assignment_id = s.assignment_id)
WHEN MATCHED THEN UPDATE SET full_score=s.full_score, score=s.score, updated_at=SYSUTCDATETIME()
WHEN NOT MATCHED THEN INSERT(assignment_id, full_score, score) VALUES(s.assignment_id, s.full_score, s.score);`,
		a.ID, in.TimeAttendance.FullScore, in.TimeAttendance.Score,
	)
	if err != nil {
		return 0, err
	}

	/* ---- Development Plan (replace all) ---- */
	if _, err = tx.ExecContext(ctx, `DELETE FROM dbo.eval_dev_plan WHERE assignment_id=@p1;`, a.ID); err != nil {
		return 0, err
	}
	if len(in.DevelopmentPlan) > 0 {
		stmtD, err2 := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_dev_plan(assignment_id, idx, content, priority, timing, remarks)
VALUES(@p1,@p2,@p3,@p4, NULLIF(@p5,''), NULLIF(@p6,''));`)
		if err2 != nil {
			return 0, err2
		}
		defer stmtD.Close()
		for _, d := range in.DevelopmentPlan {
			if _, err = stmtD.ExecContext(ctx, a.ID, d.Idx, d.Content, d.Priority, d.Timing, d.Remarks); err != nil {
				return 0, err
			}
		}
	}

	/* ---- Additional (upsert) ---- */
	_, err = tx.ExecContext(ctx, `
MERGE dbo.eval_additional AS t
USING (SELECT @p1 AS assignment_id, @p2 AS q1, @p3 AS q2, @p4 AS q3, @p5 AS q4, @p6 AS q5) s
ON (t.assignment_id = s.assignment_id)
WHEN MATCHED THEN UPDATE SET q1=s.q1, q2=s.q2, q3=s.q3, q4=s.q4, q5=s.q5, updated_at=SYSUTCDATETIME()
WHEN NOT MATCHED THEN INSERT(assignment_id, q1, q2, q3, q4, q5) VALUES(s.assignment_id, s.q1, s.q2, s.q3, s.q4, s.q5);`,
		a.ID, in.Additional.Q1, in.Additional.Q2, in.Additional.Q3, in.Additional.Q4, in.Additional.Q5,
	)
	if err != nil {
		return 0, err
	}

	/* ---- สถานะ draft/submit ---- */
	status := 0
	if in.Status == SaveSubmit {
		status = 1
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE dbo.eval_assignment SET status=@p2, updated_at=SYSUTCDATETIME() WHERE id=@p1;`,
		a.ID, status,
	); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return a.ID, nil
}

// Summary หลังบันทึก
func (r *Repo) ComputeSummary(ctx context.Context, formID, userID int) (EvalSummary, error) {
	var out EvalSummary
	out.FormID = formID

	// assignment id
	var aid int
	if err := r.DB.QueryRowContext(ctx,
		`SELECT id FROM dbo.eval_assignment WHERE form_id=@p1 AND user_id=@p2;`,
		formID, userID,
	).Scan(&aid); err != nil {
		if err == sql.ErrNoRows {
			return out, nil
		}
		return out, err
	}

	// weights ของฟอร์ม
	if err := r.DB.QueryRowContext(ctx,
		`SELECT kpi_weight, comp_weight, ta_weight FROM dbo.eval_form WHERE id=@p1;`,
		formID,
	).Scan(&out.KPIWeight, &out.CompWeight, &out.TAWeight); err != nil {
		return out, err
	}

	// KPI %
	if err := r.DB.QueryRowContext(ctx, `
SELECT ISNULL(
    (SUM(CASE WHEN ku.max_score>0 THEN (ku.score/ku.max_score) * ku.weight ELSE 0 END) 
     / NULLIF(SUM(ku.weight),0)) * 100, 0)
FROM dbo.eval_kpi_user ku 
WHERE ku.assignment_id=@p1;`, aid).Scan(&out.KPIPct); err != nil {
		return out, err
	}

	// Competency %
	if err := r.DB.QueryRowContext(ctx, `
SELECT ISNULL(SUM(
    (CASE WHEN c.max_score>0 THEN (cs.score/c.max_score) ELSE 0 END) *
    (CASE
        WHEN c.weight IS NULL THEN 0
        WHEN c.weight <= 1 THEN c.weight       -- เก็บเป็นอัตราส่วน 0..1
        ELSE c.weight/100.0                    -- เก็บเป็นเปอร์เซ็นต์ 0..100
    END)
)*100, 0)
FROM dbo.eval_competency_score cs
JOIN dbo.eval_competency c ON c.id = cs.comp_id
WHERE cs.assignment_id=@p1;`, aid).Scan(&out.CompPct); err != nil {
		return out, err
	}

	// TA %
	if err := r.DB.QueryRowContext(ctx, `
SELECT ISNULL((CASE WHEN full_score>0 THEN (score/full_score)*100 ELSE 0 END), 0)
FROM dbo.eval_ta_score WHERE assignment_id=@p1;`, aid).Scan(&out.TAPct); err != nil {
		return out, err
	}

	// รวมถ่วงน้ำหนักระดับฟอร์ม
	out.TotalPct = (out.KPIPct*float64(out.KPIWeight) +
		out.CompPct*float64(out.CompWeight) +
		out.TAPct*float64(out.TAWeight)) / 100.0

	// หาเกรด (ถ้ามี band)
	var g, has sql.NullString
	var gmin, gmax sql.NullFloat64
	err := r.DB.QueryRowContext(ctx, `
SELECT TOP(1) grade, CAST(min_pct AS float), CAST(max_pct AS float), 'x'
FROM dbo.eval_grade_band
WHERE (form_id=@p1 OR form_id IS NULL)
  AND @p2 BETWEEN min_pct AND max_pct
ORDER BY CASE WHEN form_id=@p1 THEN 0 ELSE 1 END, min_pct DESC;`,
		formID, out.TotalPct,
	).Scan(&g, &gmin, &gmax, &has)
	if err == nil {
		if g.Valid {
			out.Grade = g.String
		}
		if gmin.Valid {
			out.GradeMin = gmin.Float64
		}
		if gmax.Valid {
			out.GradeMax = gmax.Float64
		}
	}
	if out.Grade == "" {
		out.Grade = "N/A"
	}

	return out, nil
}

// ก่อน: func (r *Repo) LoadMyFormData(ctx context.Context, formID, userID int) (LoadMyFormData, error)
func (r *Repo) LoadMyFormData(ctx context.Context, formID, userID int) (LoadMyFormData, error) {
	// ทำให้ slice เป็น [] แทนที่จะเป็น nil -> JSON จะออกเป็น []
	out := LoadMyFormData{
		KPIs:            make([]MyKPIItem, 0),
		Competencies:    make([]MyCompWithScore, 0),
		DevelopmentPlan: make([]DevPlanItemInput, 0),
		TimeAttendance:  TAScoreInput{},    // 0,0
		Additional:      AdditionalInput{}, // "" ทั้ง 5 ช่อง (ไม่มี omitempty)
	}

	// หา assignment (ถ้าไม่เคยบันทึกมาก่อน จะไม่มีแถว)
	var aid, status int
	var due sql.NullString
	err := r.DB.QueryRowContext(ctx, `
        SELECT id, status, CONVERT(varchar(10),due_date,23)
        FROM dbo.eval_assignment
        WHERE form_id=@p1 AND user_id=@p2;`, formID, userID).
		Scan(&aid, &status, &due)

	if err == sql.ErrNoRows {
		// ยังไม่เคยประเมิน -> ส่ง []/ค่าว่าง ทั้งหมดกลับไปได้เลย
		return out, nil
	}
	if err != nil {
		return out, err
	}

	out.AssignmentID = aid
	out.Status = status
	if due.Valid {
		out.DueDate = due.String
	}

	// --- ดึงข้อมูลจริงต่อเมื่อมี assignment แล้ว ---
	// KPIs ของฉัน
	kpis, err := r.ListMyKPIsByForm(ctx, formID, userID)
	if err != nil {
		return out, err
	}
	out.KPIs = kpis

	// Competency + คะแนน (LEFT JOIN)
	rows, err := r.DB.QueryContext(ctx, `
SELECT c.id, c.idx, c.title, c.max_score, c.weight, c.full_total, c.expected_score,
       ISNULL(cs.score,0), ISNULL(cs.note,'')
FROM dbo.eval_competency c
LEFT JOIN dbo.eval_competency_score cs
  ON cs.comp_id = c.id AND cs.assignment_id = @p2
WHERE c.form_id=@p1
ORDER BY c.idx, c.id;`, formID, aid)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var m MyCompWithScore
		if err := rows.Scan(&m.CompID, &m.Idx, &m.Title, &m.MaxScore, &m.Weight, &m.FullTotal, &m.ExpectedScore, &m.Score, &m.Note); err != nil {
			return out, err
		}
		out.Competencies = append(out.Competencies, m)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	// Time Attendance
	_ = r.DB.QueryRowContext(ctx, `SELECT ISNULL(full_score,0), ISNULL(score,0) FROM dbo.eval_ta_score WHERE assignment_id=@p1;`, aid).
		Scan(&out.TimeAttendance.FullScore, &out.TimeAttendance.Score)

	// Dev plan
	rows, err = r.DB.QueryContext(ctx, `
SELECT idx, content, priority, CONVERT(varchar(10),timing,23), ISNULL(remarks,'')
FROM dbo.eval_dev_plan WHERE assignment_id=@p1
ORDER BY idx, id;`, aid)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var d DevPlanItemInput
		if err := rows.Scan(&d.Idx, &d.Content, &d.Priority, &d.Timing, &d.Remarks); err != nil {
			return out, err
		}
		out.DevelopmentPlan = append(out.DevelopmentPlan, d)
	}

	// Additional
	var q1, q2, q3, q4, q5 sql.NullString
	_ = r.DB.QueryRowContext(ctx, `SELECT q1,q2,q3,q4,q5 FROM dbo.eval_additional WHERE assignment_id=@p1;`, aid).
		Scan(&q1, &q2, &q3, &q4, &q5)
	if q1.Valid {
		out.Additional.Q1 = q1.String
	}
	if q2.Valid {
		out.Additional.Q2 = q2.String
	}
	if q3.Valid {
		out.Additional.Q3 = q3.String
	}
	if q4.Valid {
		out.Additional.Q4 = q4.String
	}
	if q5.Valid {
		out.Additional.Q5 = q5.String
	}

	// Summary
	sum, err := r.ComputeSummary(ctx, formID, userID)
	if err == nil {
		out.Summary = sum
	}

	return out, nil
}

// AddEvalSteps: คนแรก status=1, ที่เหลือ 0
func (r *Repo) AddEvalSteps(ctx context.Context, assignmentID int, evaluators []int) ([]EvalStep, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dbo.eval_step (assignment_id, idx, evaluator_id, status, created_at, updated_at)
OUTPUT inserted.id,
       inserted.assignment_id,
       inserted.idx,
       inserted.evaluator_id,
       inserted.status,
       CONVERT(varchar(10), inserted.eval_date, 23)
VALUES (@p1, @p2, @p3, @p4, SYSUTCDATETIME(), SYSUTCDATETIME());
`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var out []EvalStep
	for i, ev := range evaluators {
		st := 0
		if i == 0 {
			st = 1 // คนแรก active เสมอ
		}

		var s EvalStep
		if err = stmt.QueryRowContext(ctx, assignmentID, i+1, ev, st).
			Scan(&s.ID, &s.AssignmentID, &s.Idx, &s.EvaluatorID, &s.Status, &s.EvalDate); err != nil {
			return nil, err
		}

		// เติมชื่อผู้ประเมิน (ใช้ tx ให้คง transaction เดียวกัน)
		_ = tx.QueryRowContext(ctx, `SELECT name FROM dbo.users WHERE id=@p1;`, s.EvaluatorID).
			Scan(&s.EvaluatorName)

		out = append(out, s)
	}

	err = tx.Commit()
	return out, err
}

func (r *Repo) ListEvalSteps(ctx context.Context, formID, userID int) ([]EvalStep, error) {
	const q = `
SELECT es.id, es.assignment_id, es.idx,
       es.evaluator_id, u.name,
       es.status, CONVERT(varchar(10), es.eval_date, 23)
FROM dbo.eval_step es
JOIN dbo.eval_assignment a ON a.id = es.assignment_id
JOIN dbo.users u ON u.id = es.evaluator_id
WHERE a.form_id=@p1 AND a.user_id=@p2
ORDER BY es.idx;`
	rows, err := r.DB.QueryContext(ctx, q, formID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EvalStep
	for rows.Next() {
		var s EvalStep
		if err := rows.Scan(&s.ID, &s.AssignmentID, &s.Idx,
			&s.EvaluatorID, &s.EvaluatorName,
			&s.Status, &s.EvalDate); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateEvalStep: ถ้าเปลี่ยนเป็น 2=done → เปิด idx ถัดไปเป็น 1 (ถ้ายังไม่มี active)
func (r *Repo) UpdateEvalStep(ctx context.Context, stepID int, status int, evalDate string) (EvalStep, bool, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return EvalStep{}, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// lock แถวนี้ก่อน (กันชนกัน) และดึง assignment_id, idx
	var assignmentID, idx int
	if err = tx.QueryRowContext(ctx, `
SELECT assignment_id, idx
FROM dbo.eval_step WITH (UPDLOCK, ROWLOCK)
WHERE id=@p1;
`, stepID).Scan(&assignmentID, &idx); err != nil {
		if err == sql.ErrNoRows {
			return EvalStep{}, false, nil
		}
		return EvalStep{}, false, err
	}

	// อัปเดตสถานะ + eval_date แล้ว OUTPUT ค่าที่ต้องการ
	var s EvalStep
	if err = tx.QueryRowContext(ctx, `
UPDATE es
SET es.status=@p2,
    es.eval_date=NULLIF(@p3,''),
    es.updated_at=SYSUTCDATETIME()
OUTPUT inserted.id, inserted.assignment_id, inserted.idx,
       inserted.evaluator_id, u.name,
       inserted.status, CONVERT(varchar(10), inserted.eval_date, 23)
FROM dbo.eval_step es
JOIN dbo.users u ON u.id = es.evaluator_id
WHERE es.id=@p1;
`, stepID, status, evalDate).
		Scan(&s.ID, &s.AssignmentID, &s.Idx, &s.EvaluatorID, &s.EvaluatorName, &s.Status, &s.EvalDate); err != nil {
		if err == sql.ErrNoRows {
			return EvalStep{}, false, nil
		}
		return EvalStep{}, false, err
	}

	// ถ้าจบ → เปิดคนถัดไป (idx+1) เป็น 1 เฉพาะเมื่อไม่มี active อยู่แล้ว
	if status == 2 {
		var activeCnt int
		if err = tx.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM dbo.eval_step WITH (UPDLOCK, ROWLOCK)
WHERE assignment_id=@p1 AND status=1;
`, assignmentID).Scan(&activeCnt); err != nil {
			return EvalStep{}, false, err
		}

		if activeCnt == 0 {
			// เปิดเฉพาะแถวถัดไปที่ยังเป็น 0
			if _, err = tx.ExecContext(ctx, `
UPDATE dbo.eval_step
SET status=1, updated_at=SYSUTCDATETIME()
WHERE assignment_id=@p1 AND idx=@p2 AND status=0;
`, assignmentID, idx+1); err != nil {
				return EvalStep{}, false, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return EvalStep{}, false, err
	}
	return s, true, nil
}
