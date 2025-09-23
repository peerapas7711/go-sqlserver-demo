package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

func hash(pw string) ([]byte, error) {
	if strings.TrimSpace(pw) == "" {
		return nil, nil
	}
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

func (r *Repo) Create(ctx context.Context, in CreateUserInput) (User, error) {
	pwHash, err := hash(in.Password)
	if err != nil {
		return User{}, err
	}

	const q = `
INSERT INTO dbo.users
(name, email, role, password_hash, person_code, position, department, url_image, start_date, confirm_date, years_of_work, created_at, updated_at)
OUTPUT inserted.id, inserted.name, inserted.email, inserted.role, inserted.person_code, inserted.position, inserted.department, inserted.url_image, inserted.start_date, inserted.confirm_date, inserted.years_of_work, inserted.created_at, inserted.updated_at
VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8,@p9,@p10,0,SYSUTCDATETIME(),SYSUTCDATETIME());
`

	var u User
	err = r.DB.QueryRowContext(ctx, q,
		in.Name, in.Email, in.Role, pwHash,
		in.PersonCode, in.Position, in.Department,
		in.UrlImage, in.StartDate, in.ConfirmDate,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role,
		&u.PersonCode, &u.Position, &u.Department,
		&u.UrlImage, &u.StartDate, &u.ConfirmDate,
		&u.YearsOfWork, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	if len(in.CompanyIDs) > 0 {
		for _, cid := range in.CompanyIDs {
			_, _ = r.DB.ExecContext(ctx,
				"INSERT INTO dbo.user_companies(user_id, company_id) VALUES(@p1,@p2);",
				u.ID, cid,
			)
		}
	}

	return r.GetByID(ctx, u.ID)
}

func (r *Repo) GetByID(ctx context.Context, id int) (User, error) {
	const q = `
SELECT id, name, email, role, person_code, position, department, url_image,
       start_date, confirm_date, years_of_work, created_at, updated_at
FROM dbo.users WHERE id=@p1;
`
	var u User
	err := r.DB.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role,
		&u.PersonCode, &u.Position, &u.Department,
		&u.UrlImage, &u.StartDate, &u.ConfirmDate,
		&u.YearsOfWork, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	// companies
	const qc = `
SELECT c.id, c.code, c.name, c.image
FROM dbo.companies c
JOIN dbo.user_companies uc ON uc.company_id=c.id
WHERE uc.user_id=@p1;
`
	rows, err := r.DB.QueryContext(ctx, qc, id)
	if err != nil {
		return u, err
	}
	defer rows.Close()
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Image); err != nil {
			return u, err
		}
		u.Company = append(u.Company, c)
	}
	return u, nil
}

func (r *Repo) List(ctx context.Context, limit, offset int) ([]User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
SELECT id, name, email, role, person_code, position, department, url_image,
       start_date, confirm_date, years_of_work, created_at, updated_at
FROM dbo.users
ORDER BY id DESC
OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY;
`
	rows, err := r.DB.QueryContext(ctx, q, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Email, &u.Role,
			&u.PersonCode, &u.Position, &u.Department,
			&u.UrlImage, &u.StartDate, &u.ConfirmDate,
			&u.YearsOfWork, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}

		companies, _, _ := r.ListCompaniesByUser(ctx, u.ID)
		u.Company = companies

		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, id int, in UpdateUserInput) (User, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	name := u.Name
	email := u.Email
	role := u.Role
	personCode := u.PersonCode
	position := u.Position
	department := u.Department
	urlImage := u.UrlImage
	startDate := u.StartDate
	confirmDate := u.ConfirmDate
	var pwHash []byte = nil

	if in.Name != nil {
		name = *in.Name
	}
	if in.Email != nil {
		email = *in.Email
	}
	if in.Role != nil {
		role = *in.Role
	}
	if in.PersonCode != nil {
		personCode = *in.PersonCode
	}
	if in.Position != nil {
		position = *in.Position
	}
	if in.Department != nil {
		department = *in.Department
	}
	if in.UrlImage != nil {
		urlImage = *in.UrlImage
	}
	if in.StartDate != nil {
		startDate = *in.StartDate
	}
	if in.ConfirmDate != nil {
		confirmDate = *in.ConfirmDate
	}
	if in.Password != nil && strings.TrimSpace(*in.Password) != "" {
		pwHash, err = hash(*in.Password)
		if err != nil {
			return User{}, err
		}
	}

	const q = `
UPDATE dbo.users
SET name=@p1, email=@p2, role=@p3,
    person_code=@p4, position=@p5, department=@p6, url_image=@p7,
    start_date=@p8, confirm_date=@p9,
    password_hash=COALESCE(@p10, password_hash),
    updated_at=SYSUTCDATETIME()
WHERE id=@p11;
`
	_, err = r.DB.ExecContext(ctx, q,
		name, email, role,
		personCode, position, department, urlImage,
		startDate, confirmDate, pwHash, id,
	)
	if err != nil {
		return User{}, err
	}

	if in.CompanyIDs != nil {
		_, _ = r.DB.ExecContext(ctx, "DELETE FROM dbo.user_companies WHERE user_id=@p1;", id)
		for _, cid := range *in.CompanyIDs {
			_, _ = r.DB.ExecContext(ctx,
				"INSERT INTO dbo.user_companies(user_id, company_id) VALUES(@p1,@p2);",
				id, cid,
			)
		}
	}

	return r.GetByID(ctx, id)
}

func (r *Repo) Delete(ctx context.Context, id int) error {
	const q = `DELETE FROM dbo.users WHERE id=@p1;`
	res, err := r.DB.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return errors.New("not found")
	}
	return nil
}

func (r *Repo) ListCompaniesByUser(ctx context.Context, userID int) ([]Company, int, error) {
	const q = `
SELECT c.id, c.code, c.name, c.image
FROM dbo.companies c
JOIN dbo.user_companies uc ON uc.company_id=c.id
WHERE uc.user_id=@p1;
`
	rows, err := r.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Image); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, len(out), rows.Err()
}
