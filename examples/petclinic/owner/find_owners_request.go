package owner

// FindOwnersRequest binds the owner prefix and optional one-based page.
type FindOwnersRequest struct {
	LastName string `query:"lastName"`
	Page     int    `query:"page"`
	Language string `header:"Accept-Language"`
}
