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
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	bucketName := os.Getenv("GoTravelBucketName")
	basePath := "posts/"

	bucket := client.Bucket(bucketName)

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

		if attrs.Prefix == "" {
			continue
		}

		dirPrefix := attrs.Prefix
		fmt.Printf("Checking directory: %s\n", dirPrefix)

		dirIt := bucket.Objects(ctx, &storage.Query{
			Prefix:    dirPrefix,
			Delimiter: "/",
		})

		var files []string
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
				files = append(files, path.Base(fileAttrs.Name))
			}
		}

		if len(files) == 1 && files[0] == "metadata.json" {
			objPath := dirPrefix + "metadata.json"
			fmt.Printf("Deleting: %s\n", objPath)

			err := bucket.Object(objPath).Delete(ctx)
			if err != nil {
				log.Printf("Error deleting %s: %v\n", objPath, err)
			} else {
				fmt.Printf("Successfully deleted: %s\n", objPath)
			}
		}
	}

	fmt.Println("Cleanup completed!")
}
