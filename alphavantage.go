package alphavantage

import (
	"github.com/jay9909/alphavantage/net"
)

//go:generate go run cmd/apigen/main.go

type Alphavantage struct {
	client *net.Client
}

func New(apiKey string, rateLimit int, dayCap int) *Alphavantage {
	this := &Alphavantage{
		client: net.NewClient(apiKey, rateLimit, dayCap),
	}
	return this
}

func (av *Alphavantage) Close() {
	av.client.Close()
}
