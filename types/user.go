package types

import (
	"time"

	"github.com/jmoiron/sqlx/types"
)

var (
	defaultWorkDay        = time.Date(2000, 1, 1, 7, 42, 0, 0, time.UTC)
	defaultExtraWorkStart = time.Date(2000, 1, 1, 0, 19, 59, 0, time.UTC)
)

// User holds the information needed to identify each user
type User struct {
	Id             int            `db:"id"`
	FirstName      string         `db:"first_name"`
	AccessStart    time.Time      `db:"access_start_time"`
	AccessEnd      time.Time      `db:"access_end_time"`
	WorkDay        time.Time      `db:"work_day"`
	ExtraWorkStart time.Time      `db:"extra_work_start"`
	SheetId        string         `db:"sheet_id"`
	ClientSecret   types.JSONText `db:"client_secret"`
	State          State          `db:"state"`
	TimeZone       string         `db:"time_zone"`
}

// NewUser creates a new user with sensible defaults
func NewUser() *User {
	return &User{
		WorkDay:        defaultWorkDay,
		ExtraWorkStart: defaultExtraWorkStart,
		State:          Main,
	}
}
