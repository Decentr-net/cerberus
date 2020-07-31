package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

type client struct {
	host string

	pk secp256k1.PrivKeySecp256k1

	c *http.Client
}

// NewClient returns client with http.DefaultClient.
func NewClient(host string, pk secp256k1.PrivKeySecp256k1) Cerberus {
	return NewClientWithHTTPClient(host, pk, &http.Client{})
}

// NewClientWithHTTPClient returns client with provided http.Client.
func NewClientWithHTTPClient(host string, pk secp256k1.PrivKeySecp256k1, c *http.Client) Cerberus {
	return &client{
		host: host,
		pk:   pk,
		c:    c,
	}
}

// SendPDV sends bytes slice to Cerberus.
// SendPDV can return ErrInvalidRequest besides general api package's errors.
func (c *client) SendPDV(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrInvalidRequest
	}

	req := SendPDVRequest{
		Data: data,
	}
	resp := SendPDVResponse{}

	if err := c.sendRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s", c.host, SendPDVEndpoint), &req, &resp); err != nil {
		return "", fmt.Errorf("failed to make SendPDV request: %w", err)
	}

	return resp.Address, nil
}

// ReceivePDV receives bytes slice from Cerberus by provided address.
// ReceivePDV can return ErrInvalidRequest and ErrNotFound besides general api package's errors.
func (c *client) ReceivePDV(ctx context.Context, address string) ([]byte, error) {
	if address == "" {
		// nolint:godox
		return nil, ErrInvalidRequest // todo: make a strong check
	}

	req := ReceivePDVRequest{
		Address: address,
	}
	resp := ReceivePDVResponse{}

	if err := c.sendRequest(ctx, http.MethodPost, fmt.Sprintf("%s/%s", c.host, ReceivePDVEndpoint), &req, &resp); err != nil {
		return nil, fmt.Errorf("failed to make ReceivePDV request: %w", err)
	}

	return resp.Data, nil
}

// DoesPDVExist returns is data exists in Cerberus by provided address.
// DoesPDVExist can return ErrInvalidRequest and ErrNotFound besides general api package's errors.
func (c *client) DoesPDVExist(ctx context.Context, address string) (bool, error) {
	if !IsAddressValid(address) {
		return false, ErrInvalidRequest
	}

	resp := DoesPDVExistResponse{}
	url := fmt.Sprintf("%s/%s?address=%s", c.host, DoesPDVExistEndpoint, address)
	if err := c.sendRequest(ctx, http.MethodGet, url, nil, &resp); err != nil {
		return false, fmt.Errorf("failed to make DoesPDVExist request: %w", err)
	}

	return resp.Exists, nil
}

// sendRequest is utility method which signs request, if it's needed, and send POST request to Cerberus.
// Also converts http.StatusCode to package's errors.
func (c *client) sendRequest(ctx context.Context, method string, endpoint string, data interface{}, resp interface{}) error {
	if v, ok := data.(Validator); ok && !v.IsValid() {
		return ErrInvalidRequest
	}

	if err := c.signRequest(data); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s/%s", c.host, endpoint), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	rr, err := c.c.Do(r)
	if err != nil {
		return fmt.Errorf("failed to post SendPDV request: %w", err)
	}
	defer rr.Body.Close()

	if rr.StatusCode < 200 || rr.StatusCode >= 300 {
		switch rr.StatusCode {
		case http.StatusNotFound:
			return ErrNotFound
		case http.StatusBadRequest:
			return ErrInvalidRequest
		default:
			var e Error
			if err := json.NewDecoder(rr.Body).Decode(&e); err != nil {
				return errors.Errorf("request failed with status %d", rr.StatusCode)
			}
			return errors.Errorf("request failed: %s", e.Error)
		}
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *client) signRequest(r interface{}) error {
	s, ok := r.(signatureSetter)
	if !ok {
		return nil
	}

	d, err := Digest(r)
	if err != nil {
		return fmt.Errorf("failed to get digest: %w", err)
	}

	sign, err := c.pk.Sign(d)
	if err != nil {
		return fmt.Errorf("failed to sign digest: %w", err)
	}

	s.setSignature(Signature{
		PublicKey: hex.EncodeToString(c.pk.PubKey().Bytes()),
		Signature: hex.EncodeToString(sign),
	})

	return nil
}
