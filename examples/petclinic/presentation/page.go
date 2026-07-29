package presentation

// Page carries shared layout and locale state for one rendered response.
type Page struct {
	Locale string
	Active string
}

// NewPage resolves a request language and identifies the active navigation item.
func NewPage(catalog interface{ Resolve(string) string }, language, active string) Page {
	locale := "en"
	if catalog != nil {
		locale = catalog.Resolve(language)
	}
	return Page{Locale: locale, Active: active}
}
