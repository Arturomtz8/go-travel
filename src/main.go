package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

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

	subreddits := []string{
		"travel",
		"solotravel",
		"remoteplaces",
		"ruralporn",
		"rome",
		"travelphotos",
		"fujifilm",
		"travelphotography",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	randomIndex := r.Intn(len(subreddits))
	randomSubreddit := subreddits[randomIndex]
	err = reddit.GetPosts(ctx, randomSubreddit, storageService)
	if err != nil {
		log.Fatalln(err)
	}
}
