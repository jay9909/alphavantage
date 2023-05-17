package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Response struct {
	Error    error
	Response *http.Response
}

func (resp *Response) GetJson() (map[string]interface{}, error) {
	if resp.Error != nil {
		return nil, fmt.Errorf("response represents an error and cannot be parsed: %v", resp.Error)
	}

	body, err := io.ReadAll(resp.Response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	object := make(map[string]interface{})
	err = json.Unmarshal(body, &object)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON response into map: %w", err)
	}

	return object, nil
}

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
