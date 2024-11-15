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
)

const (
	timesToRecurse int = 3
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
}

type Post struct {
	Title   string
	Text    string
	Preview string
	Link    string
	Ups     int
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

			post := Post{Ups: child.Data.Ups,
				Title:   child.Data.Title,
				Text:    child.Data.SelfText,
				Preview: child.Data.UrlOverridenByDest,
				Link:    child.Data.Link,
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
	var subreddit_url string

	if iteration == timesToRecurse {
		subreddit_url = fmt.Sprintf("https://reddit.com/r/%s/.json?limit=100", subreddit)
		log.Printf("subreddit url searched: %s", subreddit_url)
	} else if iteration > 0 {
		fmt.Println("after", after)
		jsonResponse.Data.Offset = after
		subreddit_url = fmt.Sprintf("https://reddit.com/r/%s/.json?limit=100&after=%s", subreddit, jsonResponse.Data.Offset)
		log.Printf("subreddit url searched: %s", subreddit_url)
	} else {
		return childrenSliceRecursive, nil
	}

	log.Println("number of iteration", iteration)
	client := &http.Client{}
	req, err := http.NewRequest("GET", subreddit_url, nil)
	if err != nil {
		return childrenSliceRecursive, err
	}
	req.SetBasicAuth(os.Getenv("REDDIT_USER"), os.Getenv("REDDIT_PSW"))
	resp, err := client.Do(req)
	if err != nil {
		return childrenSliceRecursive, err
	}

	fmt.Println("*****************request done")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return childrenSliceRecursive, err
	}

	fmt.Println("********************read response body and trying to unmarshal")
	fmt.Println("Status", resp.Status)
	if resp.Status != "200 OK" {
		return childrenSliceRecursive, errors.New("Too many requests, try again later")
	}
	// fmt.Println("body", string(body))
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return childrenSliceRecursive, err
	}

	if len(jsonResponse.Data.Children) == 0 {
		return childrenSliceRecursive, errors.New("I couldn't get anything, make sure the subreddit exists")
	}

	for i := range jsonResponse.Data.Children {
		childrenOnly := jsonResponse.Data.Children[i]
		// log.Printf("num of times iterated: %d\n", i)
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
