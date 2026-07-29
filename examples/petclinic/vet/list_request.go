package vet

// ListRequest binds the one-based veterinarian page.
type ListRequest struct {
	Page     int    `query:"page"`
	Language string `header:"Accept-Language"`
}
