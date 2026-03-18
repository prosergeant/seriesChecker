package kinopoisk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	apiKey string
	client *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type SearchResponse struct {
	Films      []Film `json:"films"`
	PagesCount int    `json:"pagesCount"`
}

type Film struct {
	FilmID           int    `json:"filmId"`
	NameRU           string `json:"nameRu"`
	NameEN           string `json:"nameEn"`
	Year             string `json:"year"`
	PosterURL        string `json:"posterUrl"`
	PosterURLPreview string `json:"posterUrlPreview"`
}

func (f *Film) GetKinopoiskID() int {
	return f.FilmID
}

func (c *Client) Search(ctx context.Context, query string) ([]Film, error) {
	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("https://kinopoiskapiunofficial.tech/api/v2.1/films/search-by-keyword?keyword=%s&page=1", encodedQuery)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kinopoisk API returned status %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Films, nil
}

type FilmDetails struct {
	KinopoiskID int    `json:"kinopoiskId"`
	NameRU      string `json:"nameRu"`
	NameEN      string `json:"nameEn"`
	Year        int    `json:"year"`
	PosterURL   string `json:"posterUrl"`
	Description string `json:"description"`
	FilmLength  int    `json:"filmLength"`
	Series      bool   `json:"serial"`
	TotalSeries int    `json:"totalSeriesCount"`
	IsSerial    bool   `json:"is_serial"`
}

func (c *Client) GetFilm(ctx context.Context, id int) (*FilmDetails, error) {
	url := fmt.Sprintf("https://kinopoiskapiunofficial.tech/api/v2.2/films/%d", id)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kinopoisk API returned status %d", resp.StatusCode)
	}

	var result FilmDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
