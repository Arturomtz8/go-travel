package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Arturomtz8/go-travel/src/models"
	"google.golang.org/api/iterator"
)

type Post struct {
	PostID  string   `json:"post_id"`
	Title   string   `json:"title"`
	Text    string   `json:"text"`
	Link    string   `json:"reddit_link"`
	Ups     int      `json:"ups"`
	Preview string   `json:"preview"`
	GCSPath []string `json:"gcs_path"`
}

type StorageService struct {
	client     *storage.Client
	bucketName string
	ctx        context.Context
}

func NewStorageService(bucketName string) (*StorageService, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}

	s := &StorageService{
		client:     client,
		bucketName: bucketName,
		ctx:        ctx,
	}

	if err := s.ensureBucketExists(os.Getenv("GoTravelProjectID")); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %v", err)
	}

	return s, nil
}

func (s *StorageService) ensureBucketExists(projectID string) error {
	bucket := s.client.Bucket(s.bucketName)
	_, err := bucket.Attrs(s.ctx)
	if err == storage.ErrBucketNotExist {
		log.Printf("Bucket %s does not exist, creating...", s.bucketName)
		if err := bucket.Create(s.ctx, projectID, &storage.BucketAttrs{
			Location:     "US-WEST1",
			StorageClass: "STANDARD",
		}); err != nil {
			return fmt.Errorf("failed to create bucket: %v", err)
		}
		log.Printf("Bucket %s created successfully", s.bucketName)
	} else if err != nil {
		return fmt.Errorf("error checking bucket: %v", err)
	}
	return nil
}

func (s *StorageService) UploadFromURL(imageURL string, postID string, mediaID string, index int) (string, error) {
	cleanURL := cleanRedditURL(imageURL)
	req, err := http.NewRequest("GET", cleanURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	objectName := fmt.Sprintf("posts/%s/%s_%d.jpg", postID, mediaID, index)
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(s.ctx)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return "", fmt.Errorf("io.Copy: %v", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %v", err)
	}

	return fmt.Sprintf("gs://%s/%s", s.bucketName, objectName), nil
}

func (s *StorageService) PostExists(ctx context.Context, postID string) (bool, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(path.Join("posts", postID, "metadata.json"))

	_, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error checking post existence: %v", err)
	}
	return true, nil
}

func (s *StorageService) SavePost(ctx context.Context, post *models.Post) error {
	bucket := s.client.Bucket(s.bucketName)
	jsonData, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("error marshaling post data: %v", err)
	}

	obj := bucket.Object(path.Join("posts", post.PostID, "metadata.json"))
	writer := obj.NewWriter(ctx)

	if _, err := writer.Write(jsonData); err != nil {
		writer.Close()
		return fmt.Errorf("error writing post data: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing writer: %v", err)
	}

	return nil
}

func (s *StorageService) GetPost(ctx context.Context, postID string) (*Post, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(path.Join("posts", postID, "metadata.json"))

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating reader: %v", err)
	}
	defer reader.Close()

	var post Post
	if err := json.NewDecoder(reader).Decode(&post); err != nil {
		return nil, fmt.Errorf("error decoding post data: %v", err)
	}

	return &post, nil
}

func (s *StorageService) ListPosts(ctx context.Context) ([]Post, error) {
	bucket := s.client.Bucket(s.bucketName)
	query := &storage.Query{
		Prefix:    "posts/",
		Delimiter: "/",
	}

	var posts []Post
	it := bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating posts: %v", err)
		}

		if path.Base(attrs.Name) != "metadata.json" {
			continue
		}

		obj := bucket.Object(attrs.Name)
		reader, err := obj.NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("error reading post %s: %v", attrs.Name, err)
		}

		var post Post
		if err := json.NewDecoder(reader).Decode(&post); err != nil {
			reader.Close()
			return nil, fmt.Errorf("error decoding post %s: %v", attrs.Name, err)
		}
		reader.Close()

		posts = append(posts, post)
	}

	return posts, nil
}

func (s *StorageService) Close() error {
	return s.client.Close()
}

func cleanRedditURL(imageURL string) string {
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return fmt.Sprintf("error parsing URL: %v", err)
	}
	if strings.Contains(parsedURL.Host, "preview.redd.it") {
		newURL := strings.Replace(imageURL, "preview.redd.it", "i.redd.it", 1)
		return strings.Split(newURL, "?")[0]
	}
	return imageURL
}
