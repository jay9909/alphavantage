package main

import (
	"fmt"
	"github.com/jay9909/alphavantage/cmd/apigen/gen"
	"github.com/jay9909/alphavantage/cmd/apigen/parse"
)

func main() {
	// Comment out the first line and un-comment the second to force re-generation
	previousChecksum, err := gen.GetPreviousChecksum()
	// var previousChecksum [32]byte

	endpoints, accessRecord, err := parse.FindEndpoints(previousChecksum)
	if err == parse.NoChangeError {
		fmt.Printf("No change to API documentation since previous generation")
		return
	} else if err != nil {
		panic(err)
	}

	err = gen.GenerateApi(endpoints, accessRecord)
	if err != nil {
		panic(err)
	}
}
