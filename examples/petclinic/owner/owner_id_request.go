package owner

// OwnerIDRequest binds one persisted owner identity.
type OwnerIDRequest struct {
	OwnerID  int    `path:"ownerId"`
	Language string `header:"Accept-Language"`
}
