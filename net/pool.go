package net

import (
	"fmt"
	"github.com/jay9909/alphavantage/api"
	"net/http"
	"sync"
	"time"
)

type pool struct {
	requests       chan query
	workerCount    int
	maxWorkers     int
	workerCountMux sync.RWMutex
}

type query struct {
	query  string
	answer chan api.Response
}

func newPool(rateLimit int) pool {
	reqChan := make(chan query)
	workerCount := 0
	maxWorkers := rateLimit

	return pool{
		requests:       reqChan,
		workerCount:    workerCount,
		maxWorkers:     maxWorkers,
		workerCountMux: sync.RWMutex{},
	}
}

func (p *pool) doQuery() {

	request, ok := <-p.requests
	for ok == true { // Channel is not closed.  Continue
		fmt.Printf("Sending query: %v\n", request.query)
		response, err := http.Get(request.query)
		request.answer <- api.Response{
			Response: response,
			Error:    err,
		}

		time.Sleep(1*time.Minute + 3*time.Second) // Since we're rate-limited
		request, ok = <-p.requests                // Get the next query.
	}
}

func (p *pool) sendRequest(apiQuery string) api.Response {
	p.workerCountMux.Lock()
	workerCount := p.workerCount

	if workerCount < p.maxWorkers {
		fmt.Printf("Adding worker #%v\n", workerCount+1)
		go p.doQuery()
		p.workerCount++
	}
	p.workerCountMux.Unlock()

	answerChan := make(chan api.Response)
	query := query{
		query:  apiQuery,
		answer: answerChan,
	}
	p.requests <- query
	return <-answerChan
}

func (p *pool) close() {
	close(p.requests)
	p.workerCount = 0
}
