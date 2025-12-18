package email

type WelcomeEmailData struct {
	UserName  string
	UserEmail string
}

type PasswordResetEmailData struct {
	ResetLink         string
	ExpirationMinutes int
}

type PasswordUpdatedEmailData struct {
	UserName string
}
