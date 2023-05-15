package main

import (
	"alphavantage/cmd/apigen/gen"
	"alphavantage/cmd/apigen/parse"
	"fmt"
)

func main() {
	/////////////////////////////////////////////////////////////////////////////////////////////////////
	// Don't turn this on until we're actually ready to start relying on the generated code.
	//
	// previousChecksum, _ := gen.GetPreviousChecksum()
	/////////////////////////////////////////////////////////////////////////////////////////////////////
	var previousChecksum [32]byte

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
