package model

// Proxy is an entry in the administrator-managed proxy catalog.
type Proxy struct {
	ID   int64
	Name string
	URL  string
}
