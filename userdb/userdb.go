package userdb

import (
	"database/sql"

	"github.com/jmoiron/modl"
	"github.com/lnovara/workbot/types"
	// Initialize sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

var (
	dbMap *modl.DbMap
)

// NewUserDB initializes a new database to hold users informations.
func NewUserDB(dbFilePath string) error {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return err
	}

	dbMap = modl.NewDbMap(db, modl.SqliteDialect{})

	dbMap.AddTableWithName(types.User{}, "users").SetKeys(false, "Id")

	err = dbMap.CreateTablesIfNotExists()

	return err
}

// GetUser retrieves a user from a userdb
func GetUser(id int) (*types.User, error) {
	user := &types.User{}
	err := dbMap.Get(user, id)
	return user, err
}

// InsertUser inserts a new user in a userdb
func InsertUser(user *types.User) error {
	err := dbMap.Insert(user)
	return err
}

// UpdateUser updates a user in a userdb
func UpdateUser(user *types.User) error {
	_, err := dbMap.Update(user)
	return err
}

// DeleteUser delets a user in a userdb
func DeleteUser(user *types.User) error {
	_, err := dbMap.Delete(user)
	return err
}
