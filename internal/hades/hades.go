// Package hades contains code for interacting with Hades - antifraud service.
package hades

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Decentr-net/cerberus/pkg/schema"
)

var _ Hades = &client{}

//go:generate mockgen -destination=./mock/hades.go -package=mock -source=hades.go

// Hades is an interface for antifraud checking.
type Hades interface {
	AntiFraud(ctx context.Context, req *AntiFraudRequest) (*AntiFraudResponse, error)
}

// AntiFraudRequest ...
type AntiFraudRequest struct {
	ID      uint64            `json:"id"`
	Address string            `json:"address"`
	Data    schema.PDVWrapper `json:"data"`
}

// AntiFraudResponse ...
type AntiFraudResponse struct {
	IsFraud bool `json:"isFraud"`
}

// client encapsulates Hades HTTP client.
type client struct {
	baseURL string
	client  *http.Client
}

// New create a new Hades client.
func New(baseURL string) Hades {
	return &client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// AntiFraud check the given PDV for fraud.
func (c *client) AntiFraud(ctx context.Context, r *AntiFraudRequest) (*AntiFraudResponse, error) {
	path, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}

	path.Path = "/v1/pdv"

	reqBody := new(bytes.Buffer)
	if err := json.NewEncoder(reqBody).Encode(r); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx,
		http.MethodPost, path.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close() // nolint

	if httpResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", httpResp.StatusCode, string(body))
	}

	var resp AntiFraudResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode body: %w", err)
	}

	return &resp, nil
}
