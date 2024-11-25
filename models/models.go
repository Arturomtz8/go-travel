package models

type Post struct {
	PostID  string
	Title   string
	Text    string
	Preview string
	Link    string
	Ups     int
	GCSPath []string
}
