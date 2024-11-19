package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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
	Title       string
	Text        string
	Preview     string
	Link        string
	Ups         int
	LocalImages []string
}

// the slice that will hold the recursive calls, at the beginning always set it to nil
// because it can have the results from previous queries
var childrenSliceRecursive []PostSlice

func main() {
	_, err := getPosts("travel")
	if err != nil {
		log.Fatalln(err)
	}
}

func downloadImage(imageUrl, filepath string) error {
	var cleanUrl string

	parsedURL, err := url.Parse(imageUrl)
	if err != nil {
		return fmt.Errorf("error parsing URL: %v", err)
	}

	if strings.Contains(parsedURL.Host, "preview.redd.it") {
		newURL := strings.Replace(imageUrl, "preview.redd.it", "i.redd.it", 1)
		newURL = strings.Split(newURL, "?")[0]
		cleanUrl = newURL
		log.Printf("Modified URL to: %s", cleanUrl)
	}

	req, err := http.NewRequest("GET", cleanUrl, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error copying content: %v", err)
	}
	return nil
}

func getPosts(subreddit string) ([]Post, error) {
	var postsSlice []Post
	currentTime := time.Now()
	lastTwoMonths := currentTime.AddDate(0, 0, -180)

	childrenSlice, err := makeRequest(subreddit, "no", timesToRecurse)
	if err != nil {
		return nil, err
	}
	log.Println("slice len of children", len(childrenSlice))
	for _, child := range childrenSlice {
		postScore := child.Data.Ups
		createdDateUnix := child.Data.Created
		createdDate := time.Time(time.Unix(int64(createdDateUnix), 0))

		if postScore >= minPostScore && inTimeSpan(lastTwoMonths, currentTime, createdDate) {
			// log.Println(createdDate)
			child.Data.Link = "https://reddit.com" + child.Data.Link

			var localImages []string

			if child.Data.IsGallery {
				os.MkdirAll("images", 0755)
				for i, item := range child.Data.GalleryData.Items {
					if metadata, ok := child.Data.MediaMetadata[item.MediaID]; ok {
						imgURL := metadata.S.U
						filename := fmt.Sprintf("images/%s_%d.jpg", item.MediaID, i)
						err := downloadImage(imgURL, filename)
						if err != nil {
							log.Printf("Failed to download image %s: %v", imgURL, err)
							continue
						}
						localImages = append(localImages, filename)
					}
				}
			}

			post := Post{Ups: child.Data.Ups,
				Title:       child.Data.Title,
				Text:        child.Data.SelfText,
				Preview:     child.Data.UrlOverridenByDest,
				Link:        child.Data.Link,
				LocalImages: localImages,
			}
			postsSlice = append(postsSlice, post)
		}
	}
	if len(postsSlice) == 0 {
		err := errors.New("No interesting posts in subreddit")
		return nil, err
	}
	return postsSlice, nil
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

	log.Println("number of iteration", iteration)
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
	log.Printf("inside recursion len of slice after append and into another recursion: %d\n", len(childrenSliceRecursive))
	makeRequest(subreddit, jsonResponse.Data.Offset, iteration-1)
	return childrenSliceRecursive, nil

}

func inTimeSpan(lastTwoMonths, currentTime, check time.Time) bool {
	return check.After(lastTwoMonths) && check.Before(currentTime)
}
