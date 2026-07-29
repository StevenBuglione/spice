package owner

// NewOwnerRequest binds browser language preferences for owner creation.
type NewOwnerRequest struct {
	Language string `header:"Accept-Language"`
}
