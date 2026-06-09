// Package payments contains thin clients for payment providers.
// The Paystack client handles transaction initialization; webhook signature
// verification is in the handler that owns the HTTP body.
package payments

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const paystackBaseURL = "https://api.paystack.co"

// Paystack is a minimal wrapper around the subset of the Paystack REST API
// FrameLane uses.
type Paystack struct {
	Secret string
	HTTP   *http.Client
}

func NewPaystack(secret string) *Paystack {
	return &Paystack{
		Secret: strings.TrimSpace(secret),
		HTTP:   &http.Client{Timeout: 15 * time.Second},
	}
}

// InitializeRequest is the body for /transaction/initialize.
// Amount is in kobo (NGN * 100).
type InitializeRequest struct {
	Email     string            `json:"email"`
	Amount    int               `json:"amount"` // kobo
	Reference string            `json:"reference,omitempty"`
	Currency  string            `json:"currency,omitempty"`
	Callback  string            `json:"callback_url,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type InitializeResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

// Initialize creates a Paystack transaction and returns the redirect/access
// info the frontend uses to open the checkout popup or hosted page.
func (p *Paystack) Initialize(req InitializeRequest) (*InitializeResponse, error) {
	if p == nil || p.Secret == "" {
		return nil, errors.New("paystack secret not configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, paystackBaseURL+"/transaction/initialize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.Secret)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paystack initialize failed: %s: %s", resp.Status, string(raw))
	}
	var out InitializeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, errors.New(out.Message)
	}
	return &out, nil
}

// VerifyResponse is the subset of /transaction/verify/:reference we care about.
type VerifyResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID              int64             `json:"id"`
		Status          string            `json:"status"`
		Reference       string            `json:"reference"`
		Amount          int               `json:"amount"`   // kobo
		Currency        string            `json:"currency"`
		Channel         string            `json:"channel"`
		PaidAt          string            `json:"paid_at"`
		GatewayResponse string            `json:"gateway_response"`
		Metadata        map[string]string `json:"metadata"`
	} `json:"data"`
}

// Verify polls Paystack for the final state of a transaction. Use it as a
// belt-and-braces in addition to the webhook (e.g. on checkout return).
func (p *Paystack) Verify(reference string) (*VerifyResponse, error) {
	if p == nil || p.Secret == "" {
		return nil, errors.New("paystack secret not configured")
	}
	req, err := http.NewRequest(http.MethodGet, paystackBaseURL+"/transaction/verify/"+reference, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Secret)
	req.Header.Set("Accept", "application/json")
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paystack verify failed: %s: %s", resp.Status, string(raw))
	}
	var out VerifyResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
