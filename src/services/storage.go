package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

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

	if err := s.ensureBucketexists(os.Getenv("GoTravelProjectID")); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %v", err)
	}
	return s, nil
}

func (s *StorageService) ensureBucketexists(projectID string) error {
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
