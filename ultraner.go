// Package ultraner is the Go SDK for Ultraner, one API for payments across
// Africa: mobile money, cards, PayPal and wallets.
//
// Docs: https://ultraner.com/docs  ·  Spec: https://ultraner.com/openapi.json
package ultraner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.ultraner.com"

// Error is returned for non-2xx API responses.
type Error struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Status  int    `json:"-"`
}

func (e *Error) Error() string { return fmt.Sprintf("ultraner: %s (%s, %d)", e.Message, e.Code, e.Status) }

// Client is an Ultraner API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// New creates a Client with the given API key.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Do performs a request against the API. out may be nil.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// Ultraner API keys authenticate via X-API-Key (Authorization: Bearer is
	// reserved for user JWTs and would be rejected for a uk_ key).
	req.Header.Set("X-API-Key", c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	if res.StatusCode >= 300 {
		e := &Error{Status: res.StatusCode, Message: "request failed", Code: "ERROR"}
		_ = json.Unmarshal(data, e)
		e.Status = res.StatusCode
		return e
	}
	if out != nil && len(data) > 0 {
		// unwrap { "data": ... } envelope when present
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(data, &env) == nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return json.Unmarshal(data, out)
	}
	return nil
}

// PaymentResponse is the common shape for payment-like operations.
type PaymentResponse struct {
	Reference string `json:"reference"`
	Status    string `json:"status"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Provider  string `json:"provider,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// MobileMoneyCharge charges a mobile-money wallet.
type MobileMoneyCharge struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Provider      string `json:"provider"`
	AccountNumber string `json:"accountNumber"`
	ExternalID    string `json:"externalId,omitempty"`
}

// CreateMobileMoney charges a mobile-money wallet.
func (c *Client) CreateMobileMoney(ctx context.Context, in MobileMoneyCharge) (*PaymentResponse, error) {
	var out PaymentResponse
	err := c.Do(ctx, http.MethodPost, "/v1/payments/express/mno", in, &out)
	return &out, err
}

// PaymentStatus fetches a payment's status by reference.
func (c *Client) PaymentStatus(ctx context.Context, reference string) (*PaymentResponse, error) {
	var out PaymentResponse
	err := c.Do(ctx, http.MethodGet, "/v1/payments/express/status/"+url.PathEscape(reference), nil, &out)
	return &out, err
}

// Disbursement sends money to a mobile wallet or bank.
type Disbursement struct {
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Provider      string `json:"provider"`
	AccountNumber string `json:"accountNumber"`
	ExternalID    string `json:"externalId,omitempty"`
}

// CreateDisbursement sends a payout.
func (c *Client) CreateDisbursement(ctx context.Context, in Disbursement) (*PaymentResponse, error) {
	var out PaymentResponse
	err := c.Do(ctx, http.MethodPost, "/v1/disbursements", in, &out)
	return &out, err
}

// Wallet returns the wallet balances as a generic map.
func (c *Client) Wallet(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.Do(ctx, http.MethodGet, "/v1/wallet", nil, &out)
	return out, err
}

// Transactions lists transactions.
func (c *Client) Transactions(ctx context.Context, page, limit int) (map[string]any, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/v1/transactions"
	if e := q.Encode(); e != "" {
		path += "?" + e
	}
	var out map[string]any
	err := c.Do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// Escrow input.
type Escrow struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Recipient   string `json:"recipient"`
	Description string `json:"description,omitempty"`
}

// CreateEscrow opens an escrow hold.
func (c *Client) CreateEscrow(ctx context.Context, in Escrow) (map[string]any, error) {
	var out map[string]any
	err := c.Do(ctx, http.MethodPost, "/v1/escrow", in, &out)
	return out, err
}

// ReleaseEscrow releases an escrow hold to the recipient.
func (c *Client) ReleaseEscrow(ctx context.Context, escrowCode string) (map[string]any, error) {
	var out map[string]any
	err := c.Do(ctx, http.MethodPost, "/v1/escrow/"+url.PathEscape(escrowCode)+"/release", nil, &out)
	return out, err
}

// CheckoutSessionInput creates a one-time, expiring checkout token.
type CheckoutSessionInput struct {
	Amount                int64  `json:"amount"`
	Currency              string `json:"currency,omitempty"`
	Title                 string `json:"title,omitempty"`
	Description           string `json:"description,omitempty"`
	ExpiresInMinutes      int    `json:"expires_in_minutes,omitempty"`
	IssueReceipt          bool   `json:"issue_receipt,omitempty"`
	IsRecurring           bool   `json:"is_recurring,omitempty"`
	RecurringInterval     string `json:"recurring_interval,omitempty"`
	RecurringIntervalDays int    `json:"recurring_interval_days,omitempty"`
}

// CheckoutSession is a minted checkout token: open its URL or embed the token.
type CheckoutSession struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	URL       string `json:"url"`
	EmbedURL  string `json:"embed_url"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Mode      string `json:"mode"`
	ExpiresAt string `json:"expires_at"`
	Status    string `json:"status"`
}

// CreateCheckoutSession mints a one-time checkout token, the Stripe
// checkout.sessions.create parity, without touching the dashboard.
func (c *Client) CreateCheckoutSession(ctx context.Context, in CheckoutSessionInput) (*CheckoutSession, error) {
	var out CheckoutSession
	err := c.Do(ctx, http.MethodPost, "/v0/checkout/sessions", in, &out)
	return &out, err
}

// RetrieveCheckoutSession looks up a session by its token.
func (c *Client) RetrieveCheckoutSession(ctx context.Context, token string) (map[string]any, error) {
	var out map[string]any
	err := c.Do(ctx, http.MethodGet, "/v0/pay/resolve/"+url.PathEscape(token), nil, &out)
	return out, err
}
