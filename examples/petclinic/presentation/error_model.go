package presentation

// ErrorModel is the safe browser-facing error page model.
type ErrorModel struct {
	Page   Page
	Status int
	Title  string
	Detail string
}
