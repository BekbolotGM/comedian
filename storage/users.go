package storage

import (

	// This line is must for working MySQL database
	_ "github.com/go-sql-driver/mysql"
	"gitlab.com/team-monitoring/comedian/model"
)

// CreateUser creates standup entry in database
func (m *MySQL) CreateUser(u model.User) (model.User, error) {
	err := u.Validate()
	if err != nil {
		return u, err
	}
	res, err := m.conn.Exec(
		"INSERT INTO `users` (team_id, user_name, user_id, role, real_name) VALUES (?, ?, ?, ?, ?)",
		u.TeamID, u.UserName, u.UserID, u.Role, u.RealName,
	)
	if err != nil {
		return u, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return u, err
	}
	u.ID = id

	return u, nil
}

// SelectUser selects User entry from database
func (m *MySQL) SelectUser(userID string) (model.User, error) {
	var c model.User
	err := m.conn.Get(&c, "SELECT * FROM `users` WHERE user_id=?", userID)
	if err != nil {
		return c, err
	}
	return c, err
}

// GetUser selects User entry from database
func (m *MySQL) GetUser(id int64) (model.User, error) {
	var c model.User
	err := m.conn.Get(&c, "SELECT * FROM `users` WHERE id=?", id)
	if err != nil {
		return c, err
	}
	return c, err
}

// UpdateUser updates User entry in database
func (m *MySQL) UpdateUser(u model.User) (model.User, error) {
	_, err := m.conn.Exec(
		"UPDATE `users` SET role=?, real_name=? WHERE id=?",
		u.Role, u.RealName, u.ID,
	)
	if err != nil {
		return u, err
	}
	var i model.User
	err = m.conn.Get(&i, "SELECT * FROM `users` WHERE id=?", u.ID)
	return i, err
}

// ListUsers selects Users from database
func (m *MySQL) ListUsers() ([]model.User, error) {
	var u []model.User
	err := m.conn.Select(&u, "SELECT * FROM `users`")
	if err != nil {
		return u, err
	}
	return u, err
}

// DeleteUser deletes User entry from database
func (m *MySQL) DeleteUser(id int64) error {
	_, err := m.conn.Exec("DELETE FROM `users` WHERE id=?", id)
	return err
}
