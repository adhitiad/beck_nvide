package mux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"nvide-live/pkg/config"
)

type Client struct {
	tokenID     string
	tokenSecret string
	baseURL     string
}

func NewClient() *Client {
	cfg := config.Get()
	return &Client{
		tokenID:     cfg.MuxTokenID,
		tokenSecret: cfg.MuxTokenSecret,
		baseURL:     "https://api.mux.com/video/v1",
	}
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}

type LiveStreamResponse struct {
	Data struct {
		ID          string `json:"id"`
		StreamKey   string `json:"stream_key"`
		PlaybackIDs []struct {
			ID     string `json:"id"`
			Policy string `json:"policy"`
		} `json:"playback_ids"`
		Status string `json:"status"`
	} `json:"data"`
}

func (c *Client) CreateLiveStream() (*LiveStreamResponse, error) {
	payload := map[string]interface{}{
		"playback_policy": []string{"public"},
		"new_asset_settings": map[string]interface{}{
			"playback_policy": []string{"public"},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.baseURL+"/live-streams", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.tokenID, c.tokenSecret)
	req.Header.Set("Content-Type", "application/json")

	httpClient := NewHTTPClient()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("mux api error: %s", string(body))
	}

	var result LiveStreamResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetPlaybackURL(playbackID string) string {
	return fmt.Sprintf("https://stream.mux.com/%s.m3u8", playbackID)
}
