package concur

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Reddit struct {
	Max       int
	Subreddit string

	cursor int
	lastID string
	list   []string
}

func (r *Reddit) Fetch() ([]string, error) {
	logf("fetching subreddit:%s (%d/%d)", r.Subreddit, r.cursor, r.Max)
	if r.cursor >= r.Max {
		logf("fetched desired number of pages: %d", r.Max)
		return r.list, nil
	}

	var url string
	if r.lastID == "" {
		url = fmt.Sprintf("https://www.reddit.com/r/%s.json", r.Subreddit)
	} else {
		url = fmt.Sprintf("https://www.reddit.com/r/%s.json?after=t3_%s", r.Subreddit, r.lastID)
	}
	logf("making a new request at URL: %s", url)

	c := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logf("error occurred while creating http request: %s", err)
		return nil, err
	}

	req.Header.Set(
		"User-agent",
		"Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36",
	)

	res, err := c.Do(req)
	if err != nil {
		logf("error occurred while making http request: %s", err)
		return nil, err
	}

	type R struct {
		Data struct {
			Children []struct {
				Data struct {
					Title string `json:"title"`
					ID    string `json:"id"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	payload := new(R)
	if err := json.NewDecoder(res.Body).Decode(payload); err != nil {
		logf("error occurred while decoding json payload: %s", err)
		return nil, err
	}
	res.Body.Close()

	for _, v := range payload.Data.Children {
		r.list = append(r.list, v.Data.Title)
	}

	lp, lr := len(payload.Data.Children), len(r.list)
	logf("nodes appended to list (%d): %d", lr, lp)

	r.cursor++
	r.lastID = payload.Data.Children[lp-1].Data.ID

	return r.Fetch()
}
