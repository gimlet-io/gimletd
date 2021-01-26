package store

import (
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/store/sql"
	"github.com/russross/meddler"
)

// User gets a user by its login name
func (db *Store) User(login string) (*model.User, error) {
	stmt := sql.Stmt(db.driver, sql.SelectUserByLogin)
	data := new(model.User)
	err := meddler.QueryRow(db, data, stmt, login)
	return data, err
}

func (db *Store) Users() ([]*model.User, error) {
	stmt := sql.Stmt(db.driver, sql.SelectAllUser)
	var data []*model.User
	err := meddler.QueryAll(db, &data, stmt)
	return data, err
}

// CreateUser stores a new user in the database
func (db *Store) CreateUser(user *model.User) error {
	return meddler.Insert(db, "users", user)
}
