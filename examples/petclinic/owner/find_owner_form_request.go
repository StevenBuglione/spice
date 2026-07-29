package owner

// FindOwnerFormRequest binds browser language preferences for owner search.
type FindOwnerFormRequest struct {
	Language string `header:"Accept-Language"`
}
