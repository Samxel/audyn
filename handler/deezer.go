package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var deezerClient = &http.Client{Timeout: 15 * time.Second}

type DeezerAlbum struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
	ReleaseDate string `json:"release_date"`
	Cover       string `json:"cover"`
	Tracklist   string `json:"tracklist"`
	RecordType  string `json:"record_type"`
	NbTracks    int    `json:"nb_tracks"`
}

type DeezerAlbumDetail struct {
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
	NbTracks    int    `json:"nb_tracks"`
	Artist      struct {
		Name string `json:"name"`
	} `json:"artist"`
}

type DeezerResponse struct {
	Data []DeezerAlbum `json:"data"`
}

func SearchDeezer(q string) ([]DeezerAlbum, error) {
	apiURL := fmt.Sprintf("https://api.deezer.com/search/album?q=%s&limit=25", url.QueryEscape(q))

	resp, err := deezerClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DeezerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func GetAlbumDetail(id int) (DeezerAlbumDetail, error) {
	apiURL := fmt.Sprintf("https://api.deezer.com/album/%d", id)

	resp, err := deezerClient.Get(apiURL)
	if err != nil {
		return DeezerAlbumDetail{}, err
	}
	defer resp.Body.Close()

	var result DeezerAlbumDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeezerAlbumDetail{}, err
	}

	return result, nil
}
