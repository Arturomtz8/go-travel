package db

import (
	"database/sql"
	"fmt"

	"github.com/Arturomtz8/go-travel/models"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	*sql.DB
}

func InitDB(filepath string) (*Database, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

func (db *Database) CreateTables() error {
	postsTable := `
    CREATE TABLE IF NOT EXISTS posts (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        text TEXT,
        reddit_link TEXT,
        ups INTEGER,
        preview TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	imagesTable := `
    CREATE TABLE IF NOT EXISTS post_images (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        post_id TEXT,
        gcs_path TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (post_id) REFERENCES posts(id)
    );`

	if _, err := db.Exec(postsTable); err != nil {
		return fmt.Errorf("error creating posts table: %v", err)
	}

	if _, err := db.Exec(imagesTable); err != nil {
		return fmt.Errorf("error creating images table: %v", err)
	}

	return nil
}

func (db *Database) SavePost(post *models.Post) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO posts (id, title, text, reddit_link, ups, preview)
		VALUES (?, ?, ?, ?, ?, ?)`,
		post.PostID, post.Title, post.Text, post.Link, post.Ups, post.Preview)
	if err != nil {
		return fmt.Errorf("error saving post: %v", err)
	}

	for _, gcsPath := range post.GCSPath {
		_, err = tx.Exec(`
			INSERT INTO post_images (post_id, gcs_path)
			VALUES (?, ?)`,
			post.PostID, gcsPath)
		if err != nil {
			return fmt.Errorf("error saving image: %v", err)
		}
	}
	return tx.Commit()

}
