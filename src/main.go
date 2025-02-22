package main

import (
	"context"
	"log"
	"os"

	"github.com/Arturomtz8/go-travel/src/reddit"
	"github.com/Arturomtz8/go-travel/src/storage"
)

func main() {
	ctx := context.Background()

	// Initialize storage service
	storageService, err := storage.NewStorageService(os.Getenv("GoTravelBucketName"))
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}
	defer storageService.Close()

	// Get posts from Reddit
	err = reddit.GetPosts(ctx, "travel", storageService)
	if err != nil {
		log.Fatalln(err)
	}
}
