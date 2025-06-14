package soundcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Track struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Media struct {
		Transcodings []struct {
			URL    string `json:"url"`
			Format struct {
				Protocol string `json:"protocol"`
			} `json:"format"`
		} `json:"transcodings"`
	} `json:"media"`
	StreamURL string
}

func GetTracks(query, clientID string) ([]Track, error) {
	q := url.QueryEscape(query)
	url := fmt.Sprintf("https://api-v2.soundcloud.com/search/tracks?q=%s&client_id=%s&limit=15", q, clientID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Collection []Track `json:"collection"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	for i := range result.Collection {
		for _, t := range result.Collection[i].Media.Transcodings {
			if t.Format.Protocol == "progressive" {
				streamURL, err := ResolveStreamURL(t.URL, clientID)
				if err == nil {
					result.Collection[i].StreamURL = streamURL
					break
				}
			}
		}
	}

	var filtered []Track
	for _, t := range result.Collection {
		if t.StreamURL != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

func ResolveStreamURL(apiURL, clientID string) (string, error) {
	url := fmt.Sprintf("%s?client_id=%s", apiURL, clientID)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		URL string `json:"url"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.URL, nil
}
