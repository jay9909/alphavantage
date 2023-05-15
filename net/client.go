package net

import "fmt"

type Client struct {
	apiKey    string
	rateLimit int // Currently 5, 75, 150, 300, 600, or 1200 requests per minute
	dayCap    int // The free API tier is capped at 500 requests/day.  Paid tiers are not capped.
}

func NewClient(apiKey string, rateLimit, dayCap int) *Client {
	return &Client{
		apiKey:    apiKey,
		rateLimit: rateLimit,
		dayCap:    dayCap,
	}
}

func (c *Client) Hello() {
	fmt.Println("Hello")
}
