package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()

	// Create a client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Use the same path and bucket configuration
	basePath := "posts/"
	bucket := client.Bucket(os.Getenv("GoTravelBucketName"))

	// List all directories under basePath
	it := bucket.Objects(ctx, &storage.Query{
		Prefix:    basePath,
		Delimiter: "/",
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error iterating: %v", err)
		}

		// Skip if not a prefix (directory)
		if attrs.Prefix == "" {
			continue
		}

		dirPrefix := attrs.Prefix
		fmt.Printf("Checking directory: %s\n", dirPrefix)

		// List files in this directory
		dirIt := bucket.Objects(ctx, &storage.Query{
			Prefix:    dirPrefix,
			Delimiter: "/",
		})

		var files []string
		hasMetadata := false
		var objectsToDelete []string

		// First pass: check files and look for metadata.json
		for {
			fileAttrs, err := dirIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Error listing files in %s: %v\n", dirPrefix, err)
				break
			}
			if fileAttrs.Name != "" {
				filename := path.Base(fileAttrs.Name)
				files = append(files, filename)
				if filename == "metadata.json" {
					hasMetadata = true
				}
				objectsToDelete = append(objectsToDelete, fileAttrs.Name)
			}
		}

		// If directory doesn't have metadata.json, delete all files
		if !hasMetadata && len(files) > 0 {
			fmt.Printf("Directory %s has no metadata.json, deleting all files\n", dirPrefix)
			for _, objPath := range objectsToDelete {
				fmt.Printf("Deleting: %s\n", objPath)
				err := bucket.Object(objPath).Delete(ctx)
				if err != nil {
					log.Printf("Error deleting %s: %v\n", objPath, err)
				} else {
					fmt.Printf("Successfully deleted: %s\n", objPath)
				}
			}
		}
	}

	fmt.Println("Cleanup completed!")
}
