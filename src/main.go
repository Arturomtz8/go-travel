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

	storageService, err := storage.NewStorageService(os.Getenv("GoTravelBucketName"))
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}
	defer storageService.Close()

	err = reddit.GetPosts(ctx, "rome", storageService)
	if err != nil {
		log.Fatalln(err)
	}
}
