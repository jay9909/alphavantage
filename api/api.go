package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const freeApiLimitReached = `{
    "Note": "Thank you for using Alpha Vantage! Our standard API call frequency is 5 calls per minute and 500 calls per day. Please visit https://www.alphavantage.co/premium/ if you would like to target a higher API call frequency."
}`

type Response struct {
	Error    error
	Response *http.Response
}

// GetJson populates the provided reference with a decoded JSON response.
func (resp *Response) GetJson(result interface{}) error {
	if resp.Error != nil {
		return fmt.Errorf("response represents an error and cannot be parsed: %v", resp.Error)
	}

	body, err := io.ReadAll(resp.Response.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %w", err)
	}

	if freeApiLimitReached == string(body) {
		return fmt.Errorf("exceeded free api rate limit: %v", body)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("could not parse JSON response into map: %w\n=====%v\n=====\n",
			err, string(body))
	}

	return nil
}

// GetCsv returns the text body of the response with no modifications.
func (resp *Response) GetCsv() (string, error) {
	if resp.Error != nil {
		return "", fmt.Errorf("response represents an error and cannot be parsed: %v", resp.Error)
	}

	body, err := io.ReadAll(resp.Response.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response body: %w", err)
	}

	return string(body), nil
}

// GetText returns the text body of the response with no modifications.
func (resp *Response) GetText() (string, error) {
	return resp.GetCsv()
}
