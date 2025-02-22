package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Arturomtz8/go-travel/src/models"
	"github.com/Arturomtz8/go-travel/src/storage"
)

const (
	timesToRecurse int = 2
	minPostScore   int = 100
	minusDays      int = 120
)

// the slice that will hold the recursive calls
var childrenSliceRecursive []PostSlice

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

func GetPosts(ctx context.Context, subreddit string, storageService *storage.StorageService) error {
	currentTime := time.Now()
	pastTime := currentTime.AddDate(0, 0, -minusDays)
	postsProcessed := 0
	postsSkipped := 0

	childrenSlice, err := makeRequest(subreddit, "no", timesToRecurse)
	if err != nil {
		return err
	}

	for _, child := range childrenSlice {
		exists, err := storageService.PostExists(ctx, child.Data.ID)
		if exists {
			log.Printf("Skipping post %s: already exists in storage", child.Data.ID)
			postsSkipped++
			continue
		}

		postScore := child.Data.Ups
		createdDateUnix := child.Data.Created
		createdDate := time.Unix(int64(createdDateUnix), 0)

		if postScore >= minPostScore && inTimeSpan(pastTime, currentTime, createdDate) {
			if err != nil {
				log.Printf("Error checking post existence: %v", err)
				continue
			}

			var gcsImages []string
			child.Data.Link = "https://reddit.com" + child.Data.Link

			if child.Data.IsGallery {
				for i, item := range child.Data.GalleryData.Items {
					if metadata, ok := child.Data.MediaMetadata[item.MediaID]; ok {
						imgURL := metadata.S.U
						gcsPath, err := storageService.UploadFromURL(
							imgURL,
							child.Data.ID,
							item.MediaID,
							i,
						)
						if err != nil {
							log.Printf("Failed to upload image %s: %v", imgURL, err)
							continue
						}
						gcsImages = append(gcsImages, gcsPath)
					}
				}
			}

			post := &models.Post{
				PostID:  child.Data.ID,
				Title:   child.Data.Title,
				Text:    child.Data.SelfText,
				Link:    child.Data.Link,
				Ups:     child.Data.Ups,
				Preview: child.Data.UrlOverridenByDest,
				GCSPath: gcsImages,
			}

			if err := storageService.SavePost(ctx, post); err != nil {
				log.Printf("Failed to save post %s: %v", post.PostID, err)
				continue
			}
			postsProcessed++
		}
	}

	if postsProcessed == 0 {
		log.Printf("Didn't process any post. Posts skipped: %v", postsSkipped)
		return nil
	}

	log.Printf("Successfully processed %d new posts, skipped %d existing posts",
		postsProcessed, postsSkipped)
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

func inTimeSpan(pastTime, currentTime, check time.Time) bool {
	return check.After(pastTime) && check.Before(currentTime)
}
