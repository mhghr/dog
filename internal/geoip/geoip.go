package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Location struct {
	Country string  `json:"country"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

var client = &http.Client{Timeout: 5 * time.Second}

func Lookup(ip string) (*Location, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=country,city,lat,lon", ip)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("geoip request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geoip returned status %d", resp.StatusCode)
	}

	var loc Location
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil, fmt.Errorf("geoip decode failed: %w", err)
	}

	return &loc, nil
}
