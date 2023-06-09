package net

import (
	"fmt"
	"github.com/jay9909/alphavantage/api"
	"net/url"
	"strings"
)

const baseUrl = "https://www.alphavantage.co/query?"

type Client struct {
	apiKey    string
	rateLimit int  // Currently 5, 75, 150, 300, 600, or 1200 requests per minute
	dayCap    int  // The free API tier is capped at 500 requests/day.  Paid tiers are not capped.
	reqPool   pool // Pool of requesters.
}

func NewClient(apiKey string, rateLimit, dayCap int) *Client {
	return &Client{
		apiKey:    apiKey,
		rateLimit: rateLimit,
		dayCap:    dayCap,
		reqPool:   newPool(rateLimit),
	}
}

// Query sends the given request to the Alphavantage service.  Note: params should NOT include the function or apiKey
// parameter key/value pairs.
func (c *Client) Query(function string, params map[string]string) api.Response {
	var urlBuilder strings.Builder
	urlBuilder.WriteString(baseUrl)
	urlBuilder.WriteString(fmt.Sprintf("function=%v", function))
	urlBuilder.WriteString(fmt.Sprintf("&apikey=%v", c.apiKey))

	for key, value := range params {
		if value != "" && key != "function" && key != "apikey" {
			urlBuilder.WriteString(
				fmt.Sprintf("&%v=%v", url.QueryEscape(key), url.QueryEscape(value)))
		}
	}

	return c.reqPool.sendRequest(urlBuilder.String())
}

func (c *Client) Close() {
	c.reqPool.close()
}
