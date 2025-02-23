package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

func moveFilesToSubfolder(ctx context.Context, client *storage.Client, bucketName string) error {
	bucket := client.Bucket(os.Getenv("GoTravelBucketName"))

	query := &storage.Query{
		Prefix: "posts/",
	}

	fmt.Println("Starting to move files to posts/travel/...")

	count := 0
	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing objects: %v", err)
		}

		if strings.Contains(attrs.Name, "posts/travel/") || attrs.Name == "posts/travel/" {
			continue
		}

		oldName := attrs.Name
		newName := strings.Replace(oldName, "posts/", "posts/travel/", 1)

		src := bucket.Object(oldName)
		dst := bucket.Object(newName)

		copier := dst.CopierFrom(src)
		if _, err := copier.Run(ctx); err != nil {
			return fmt.Errorf("error copying %q to %q: %v", oldName, newName, err)
		}
		if err := src.Delete(ctx); err != nil {
			return fmt.Errorf("error deleting %q: %v", oldName, err)
		}

		count++
		fmt.Printf("Moved: %s → %s\n", oldName, newName)
	}

	fmt.Printf("\nCompleted! Moved %d items to posts/travel/\n", count)
	return nil
}

func main() {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	bucketName := os.Getenv("GoTravelBucketName")

	if err := moveFilesToSubfolder(ctx, client, bucketName); err != nil {
		log.Fatalf("Failed to move files: %v", err)
	}
}
