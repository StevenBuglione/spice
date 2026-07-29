package system

// WelcomeRequest binds browser language preferences for the home page.
type WelcomeRequest struct {
	Language string `header:"Accept-Language"`
}
