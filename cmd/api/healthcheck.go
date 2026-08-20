package main

import (
	"context"
	"net/http"
	"time"
)

func runHealthcheck() int {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://localhost:8080/api/v1/health",
		nil,
	)

	if err != nil {
		return 1
	}

	resp, err := client.Do(req)
	if err != nil {
		return 1
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}

	return 0
}
