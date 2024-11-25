package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Arturomtz8/go-travel/db"
	"github.com/Arturomtz8/go-travel/models"
	"github.com/Arturomtz8/go-travel/services"
)

const (
	timesToRecurse int = 2
	minPostScore   int = 50
)

type JSONResponse struct {
	Data Data `json:"data"`
}

type Data struct {
	Children []PostSlice `json:"children"`
	Offset   string      `json:"after"`
}

type PostSlice struct {
	Data PostData `json:"data"`
}

type PostData struct {
	ID                 string  `json:"id"`
	Ups                int     `json:"ups"`
	Title              string  `json:"title"`
	SelfText           string  `json:"selftext"`
	Link               string  `json:"permalink"`
	Created            float64 `json:"created"`
	UrlOverridenByDest string  `json:"url_overridden_by_dest"`
	IsGallery          bool    `json:"is_gallery"`
	MediaMetadata      map[string]struct {
		Status string `json:"status"`
		S      struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
	GalleryData struct {
		Items []struct {
			MediaID string `json:"media_id"`
		} `json:"items"`
	} `json:"gallery_data"`
}

type Post struct {
	PostID  string
	Title   string
	Text    string
	Preview string
	Link    string
	Ups     int
	gcsPath []string
}

// the slice that will hold the recursive calls, at the beginning always set it to nil
// because it can have the results from previous queries
var childrenSliceRecursive []PostSlice

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

	err = getPosts("travel", storageService, database)
	if err != nil {
		log.Fatalln(err)
	}
}

func getPosts(subreddit string, storageService *services.StorageService, database *db.Database) error {
	currentTime := time.Now()
	lastTwoMonths := currentTime.AddDate(0, 0, -180)
	postsProcessed := 0

	childrenSlice, err := makeRequest(subreddit, "no", timesToRecurse)
	if err != nil {
		return err
	}
	for _, child := range childrenSlice {
		postScore := child.Data.Ups
		createdDateUnix := child.Data.Created
		createdDate := time.Time(time.Unix(int64(createdDateUnix), 0))

		if postScore >= minPostScore && inTimeSpan(lastTwoMonths, currentTime, createdDate) {
			child.Data.Link = "https://reddit.com" + child.Data.Link

			var gcsImages []string

			if child.Data.IsGallery {
				for i, item := range child.Data.GalleryData.Items {
					if metadata, ok := child.Data.MediaMetadata[item.MediaID]; ok {
						imgURL := metadata.S.U
						gcsPath, err := storageService.UploadFromURL(imgURL,
							child.Data.ID,
							item.MediaID,
							i)
						if err != nil {
							log.Printf("Failed to upload image %s: %v", imgURL, err)
							continue
						}
						gcsImages = append(gcsImages, gcsPath)
					}
				}
			}

			post := models.Post{
				PostID:  child.Data.ID,
				Ups:     child.Data.Ups,
				Title:   child.Data.Title,
				Text:    child.Data.SelfText,
				Preview: child.Data.UrlOverridenByDest,
				Link:    child.Data.Link,
				GCSPath: gcsImages,
			}

			if err := database.SavePost(&post); err != nil {
				log.Printf("Failed to save post %s: %v", post.PostID, err)
				continue
			}
			postsProcessed++
		}
	}
	if postsProcessed == 0 {
		err := errors.New("No interesting posts in subreddit")
		return err
	}
	return nil
}

func makeRequest(subreddit, after string, iteration int) ([]PostSlice, error) {
	var jsonResponse JSONResponse
	var subredditUrl string

	if iteration == timesToRecurse {
		subredditUrl = fmt.Sprintf("https://reddit.com/r/%s/.json?limit=100", subreddit)
	} else if iteration > 0 {
		jsonResponse.Data.Offset = after
		subredditUrl = fmt.Sprintf("https://reddit.com/r/%s/.json?limit=100&after=%s", subreddit, jsonResponse.Data.Offset)
	} else {
		return childrenSliceRecursive, nil
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", subredditUrl, nil)
	if err != nil {
		return childrenSliceRecursive, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return childrenSliceRecursive, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return childrenSliceRecursive, err
	}

	if resp.Status != "200 OK" {
		return childrenSliceRecursive, errors.New("Too many requests, try again later")
	}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return childrenSliceRecursive, err
	}

	if len(jsonResponse.Data.Children) == 0 {
		return childrenSliceRecursive, errors.New("I couldn't get anything, make sure the subreddit exists")
	}

	for i := range jsonResponse.Data.Children {
		childrenOnly := jsonResponse.Data.Children[i]
		childrenSliceRecursive = append(childrenSliceRecursive, childrenOnly)
	}

	defer resp.Body.Close()
	makeRequest(subreddit, jsonResponse.Data.Offset, iteration-1)
	return childrenSliceRecursive, nil

}

func inTimeSpan(lastTwoMonths, currentTime, check time.Time) bool {
	return check.After(lastTwoMonths) && check.Before(currentTime)
}
