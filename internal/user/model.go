package user

import "time"

type Company struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Company     []Company `json:"company"`
	PersonCode  string    `json:"personcode"`
	Position    string    `json:"position"`
	Department  string    `json:"department"`
	UrlImage    string    `json:"urlimage"`
	StartDate   string    `json:"start_date"`
	ConfirmDate string    `json:"confirm_date"`
	YearsOfWork int       `json:"years_of_work"`
}

type CreateUserInput struct {
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Password    string   `json:"password"`
	PersonCode  string   `json:"personcode"`
	Position    string   `json:"position"`
	Department  string   `json:"department"`
	UrlImage    string   `json:"urlimage"`
	StartDate   string   `json:"start_date"`
	ConfirmDate string   `json:"confirm_date"`
	CompanyIDs  []string `json:"company_ids"`
}

type UpdateUserInput struct {
	Name        *string   `json:"name,omitempty"`
	Email       *string   `json:"email,omitempty"`
	Role        *string   `json:"role,omitempty"`
	Password    *string   `json:"password,omitempty"`
	PersonCode  *string   `json:"personcode,omitempty"`
	Position    *string   `json:"position,omitempty"`
	Department  *string   `json:"department,omitempty"`
	UrlImage    *string   `json:"urlimage,omitempty"`
	StartDate   *string   `json:"start_date,omitempty"`
	ConfirmDate *string   `json:"confirm_date,omitempty"`
	CompanyIDs  *[]string `json:"company_ids,omitempty"`
}
