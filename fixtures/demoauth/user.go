package demoauth

import "errors"

// DemoUserEmail is the only account in this fixture.
const DemoUserEmail = "dev@example.test"

// demoPassword is a fake password used only by the walkthrough.
const demoPassword = "demo-only"

// User is a demo account record. PasswordHash is not a real password hash.
type User struct {
	ID           string
	Email        string
	PasswordHash string
}

var users = map[string]User{
	DemoUserEmail: {
		ID:           "user_demo_1",
		Email:        DemoUserEmail,
		PasswordHash: "redacted-demo-hash",
	},
}

// lookupUser returns the demo user for an email address.
func lookupUser(email string) (User, error) {
	user, ok := users[email]
	if !ok {
		return User{}, errNotFound
	}
	return user, nil
}

// checkPassword compares password with the fixture password.
func checkPassword(user User, password string) bool {
	if user.Email != DemoUserEmail {
		return false
	}
	return password == demoPassword
}

var errNotFound = errors.New("user not found")
