package model

import (
	apsql "gateway/sql"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptPasswordCost = 10

// User represents a user!
type User struct {
	AccountID               int64  `json:"-" db:"account_id"`
	ID                      int64  `json:"id"`
	Name                    string `json:"name"`
	Email                   string `json:"email"`
	NewPassword             string `json:"password"`
	NewPasswordConfirmation string `json:"password_confirmation"`
	HashedPassword          string `json:"-" db:"hashed_password"`
}

// Validate validates the model.
func (u *User) Validate() Errors {
	errors := make(Errors)
	if u.Name == "" {
		errors.add("name", "must not be blank")
	}
	if u.Email == "" {
		errors.add("email", "must not be blank")
	}
	if u.ID == 0 && u.NewPassword == "" {
		errors.add("password", "must not be blank")
	}
	if u.NewPassword != "" && (u.NewPassword != u.NewPasswordConfirmation) {
		errors.add("password_confirmation", "must match password")
	}
	return errors
}

// AllUsersForAccountID returns all users on the Account in default order.
func AllUsersForAccountID(db *apsql.DB, accountID int64) ([]*User, error) {
	users := []*User{}
	err := db.Select(&users,
		`SELECT id, name, email FROM users
		 WHERE account_id = ? ORDER BY name ASC;`,
		accountID)
	return users, err
}

// FindUserForAccountID returns the user with the id and account_id specified.
func FindUserForAccountID(db *apsql.DB, id, accountID int64) (*User, error) {
	user := User{}
	err := db.Get(&user,
		`SELECT id, name, email FROM users
		 WHERE id = ? AND account_id = ?;`,
		id, accountID)
	return &user, err
}

// DeleteUserForAccountID deletes the user with the id and account_id specified.
func DeleteUserForAccountID(tx *apsql.Tx, id, accountID int64) error {
	return tx.DeleteOne(
		`DELETE FROM users
		 WHERE id = ? AND account_id = ?;`,
		id, accountID)
}

// FindUserByEmail returns the user with the email specified.
func FindUserByEmail(db *apsql.DB, email string) (*User, error) {
	user := User{}
	err := db.Get(&user,
		`SELECT id, account_id, hashed_password
		 FROM users WHERE email = ?;`,
		strings.ToLower(email))
	return &user, err
}

// Insert inserts the user into the database as a new row.
func (u *User) Insert(tx *apsql.Tx) (err error) {
	if err = u.hashPassword(); err != nil {
		return err
	}

	u.ID, err = tx.InsertOne(
		`INSERT INTO users
		        (account_id, name, email, hashed_password)
		 VALUES (?, ?, ?, ?);`,
		u.AccountID, u.Name, strings.ToLower(u.Email), u.HashedPassword)
	return err
}

// Update updates the user in the database.
func (u *User) Update(tx *apsql.Tx) error {
	var err error
	if u.NewPassword != "" {
		err = u.hashPassword()
		if err != nil {
			return err
		}
		return tx.UpdateOne(
			`UPDATE users
			 SET name = ?, email = ?, hashed_password = ?
			 WHERE id = ? AND account_id = ?;`,
			u.Name, strings.ToLower(u.Email), u.HashedPassword, u.ID, u.AccountID)
	}

	return tx.UpdateOne(
		`UPDATE users
			 SET name = ?, email = ?
			 WHERE id = ? AND account_id = ?;`,
		u.Name, strings.ToLower(u.Email), u.ID, u.AccountID)
}

func (u *User) hashPassword() error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.NewPassword), bcryptPasswordCost)
	if err != nil {
		return err
	}
	u.HashedPassword = string(hashed)
	return nil
}

// ValidPassword returns whether or not the password matches what's on file.
func (u *User) ValidPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
	return err == nil
}
