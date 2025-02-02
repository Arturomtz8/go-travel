package main

import (
	"log"
	"os"

	"github.com/Arturomtz8/go-travel/src/db"
	"github.com/Arturomtz8/go-travel/src/reddit"
	"github.com/Arturomtz8/go-travel/src/services"
)

func main() {
	storageService, err := services.NewStorageService(os.Getenv("GoTravelBucketName"))
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}

	database, err := db.InitDB("reddit_posts.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	defer database.Close()

	if err := database.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	err = reddit.GetPosts("travel", storageService, database)
	if err != nil {
		log.Fatalln(err)
	}
}
