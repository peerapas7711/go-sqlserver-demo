package timeattendance

import (
	"context"
	"database/sql"
)

type Repo struct{ DB *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

// Upsert ทั้ง summary และ items (replace ทั้งชุด)
func (r *Repo) Upsert(ctx context.Context, userID int, in Payload) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// upsert summary
	_, err = tx.ExecContext(ctx, `
MERGE dbo.user_time_attendance AS t
USING (SELECT @p1 AS user_id, @p2 AS score_max, @p3 AS score_obtained, @p4 AS penalty_total) AS s
ON (t.user_id = s.user_id)
WHEN MATCHED THEN UPDATE SET
  score_max = s.score_max,
  score_obtained = s.score_obtained,
  penalty_total = s.penalty_total
WHEN NOT MATCHED THEN
  INSERT (user_id, score_max, score_obtained, penalty_total)
  VALUES (s.user_id, s.score_max, s.score_obtained, s.penalty_total);
`, userID, in.ScoreMax, in.ScoreObtained, in.PenaltyTotal)
	if err != nil {
		return err
	}

	// ลบของเก่า แล้วใส่ชุดใหม่ (ง่ายและตรงตาม payload)
	if _, err = tx.ExecContext(ctx, `DELETE FROM dbo.user_time_attendance_items WHERE user_id=@p1;`, userID); err != nil {
		return err
	}
	if len(in.TimeAttendance) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dbo.user_time_attendance_items(user_id, name, value)
VALUES(@p1, @p2, @p3);
`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, it := range in.TimeAttendance {
			if _, err = stmt.ExecContext(ctx, userID, it.Name, it.Value); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repo) Get(ctx context.Context, userID int) (Payload, bool, error) {
	var out Payload
	// summary
	err := r.DB.QueryRowContext(ctx, `
SELECT score_max, score_obtained, penalty_total
FROM dbo.user_time_attendance WHERE user_id=@p1;
`, userID).Scan(&out.ScoreMax, &out.ScoreObtained, &out.PenaltyTotal)
	if err == sql.ErrNoRows {
		return Payload{}, false, nil
	}
	if err != nil {
		return Payload{}, false, err
	}
	// items
	rows, err := r.DB.QueryContext(ctx, `
SELECT name, value
FROM dbo.user_time_attendance_items
WHERE user_id=@p1 ORDER BY name;`, userID)
	if err != nil {
		return Payload{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.Name, &it.Value); err != nil {
			return Payload{}, false, err
		}
		out.TimeAttendance = append(out.TimeAttendance, it)
	}
	return out, true, rows.Err()
}
