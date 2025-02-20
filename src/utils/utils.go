package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"cloud.google.com/go/storage"
	_ "github.com/mattn/go-sqlite3"
)

type SQLitePost struct {
	ID         string
	Title      string
	Text       string
	RedditLink string
	Ups        int
	Preview    string
	Images     []string
}

func main() {
	ctx := context.Background()

	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}
	defer gcsClient.Close()

	db, err := sql.Open("sqlite3", "./reddit_posts.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	posts, err := getPostsFromSQLite(db)
	if err != nil {
		log.Fatalf("Failed to get posts from SQLite: %v", err)
	}

	bucket := gcsClient.Bucket(os.Getenv("GoTravelBucketName"))
	for _, post := range posts {
		if err := migratePostToGCS(ctx, bucket, post); err != nil {
			log.Printf("Error migrating post %s: %v", post.ID, err)
			continue
		}
		log.Printf("Successfully migrated post %s", post.ID)
	}
}

func getPostsFromSQLite(db *sql.DB) ([]SQLitePost, error) {
	rows, err := db.Query(`
        SELECT id, title, text, reddit_link, ups, preview 
        FROM posts
    `)
	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

	var posts []SQLitePost
	for rows.Next() {
		var post SQLitePost
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Text,
			&post.RedditLink,
			&post.Ups,
			&post.Preview,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %v", err)
		}

		post.Images, err = getImagesForPost(db, post.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get images for post %s: %v", post.ID, err)
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func getImagesForPost(db *sql.DB, postID string) ([]string, error) {
	rows, err := db.Query(`
        SELECT gcs_path 
        FROM post_images 
        WHERE post_id = ?
    `, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		images = append(images, path)
	}

	return images, nil
}

func migratePostToGCS(ctx context.Context, bucket *storage.BucketHandle, post SQLitePost) error {
	metadata := map[string]interface{}{
		"post_id":     post.ID,
		"title":       post.Title,
		"text":        post.Text,
		"reddit_link": post.RedditLink,
		"ups":         post.Ups,
		"preview":     post.Preview,
		"gcs_path":    post.Images,
	}

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	obj := bucket.Object(path.Join("posts", post.ID, "metadata.json"))
	writer := obj.NewWriter(ctx)

	if _, err := writer.Write(jsonData); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write metadata: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	return nil
}
