package model

// User is the user representation
type User struct {
	// ID for this user
	// required: true
	ID int64 `json:"-" meddler:"id,pk"`

	// Login is the username for this user
	// required: true
	Login string `json:"login"  meddler:"login"`

	// Secret is the PEM formatted RSA private key used to sign JWT and CSRF tokens
	Secret string `json:"-" meddler:"secret"`
}
