package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	APIURL         = "https://api.yookassa.ru/v3"
	requestTimeout = 5 * time.Second
)

type Client struct {
	endpoint, shopID, secret, returnURL string
	http                                *http.Client
}

type Payment struct {
	ID              string
	Status          string
	Currency        string
	Amount          decimal.Decimal
	ConfirmationURL *string
}

type paymentAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

func NewClient(endpoint, shopID, secret, returnURL string) (*Client, error) {
	target, targetErr := url.Parse(endpoint)
	if targetErr != nil || target.Hostname() == "" || (target.Scheme != "https" && target.Scheme != "http") ||
		target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return nil, errors.New("invalid YooKassa API endpoint")
	}
	callback, err := url.Parse(returnURL)
	if err != nil || callback.Hostname() == "" || (callback.Scheme != "https" && callback.Scheme != "http") ||
		callback.User != nil || callback.Fragment != "" {
		return nil, errors.New("YOOKASSA_RETURN_URL must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(shopID) == "" || strings.TrimSpace(secret) == "" {
		return nil, errors.New("YooKassa test shop credentials are required")
	}
	return &Client{endpoint: strings.TrimRight(endpoint, "/"), shopID: shopID, secret: secret, returnURL: returnURL,
		http: &http.Client{Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}}, nil
}

func (c *Client) CreatePayment(ctx context.Context, key string, amount decimal.Decimal) (Payment, error) {
	body := struct {
		Amount       paymentAmount     `json:"amount"`
		Capture      bool              `json:"capture"`
		Confirmation map[string]string `json:"confirmation"`
	}{
		Amount:       paymentAmount{Value: amount.StringFixed(2), Currency: "RUB"},
		Capture:      true,
		Confirmation: map[string]string{"type": "redirect", "return_url": c.returnURL},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Payment{}, err
	}
	return c.request(ctx, http.MethodPost, "/payments", key, bytes.NewReader(encoded))
}

func (c *Client) GetPayment(ctx context.Context, id string) (Payment, error) {
	return c.request(ctx, http.MethodGet, "/payments/"+url.PathEscape(id), "", nil)
}

func (c *Client) request(ctx context.Context, method, path, key string, body io.Reader) (Payment, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if err != nil {
		return Payment{}, err
	}
	req.SetBasicAuth(c.shopID, c.secret)
	if key != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotence-Key", key)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Payment{}, fmt.Errorf("request YooKassa: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return Payment{}, fmt.Errorf("YooKassa returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		ID           string        `json:"id"`
		Status       string        `json:"status"`
		Test         bool          `json:"test"`
		Amount       paymentAmount `json:"amount"`
		Confirmation struct {
			URL *string `json:"confirmation_url"`
		} `json:"confirmation"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return Payment{}, fmt.Errorf("decode YooKassa response: %w", err)
	}
	amount, err := decimal.NewFromString(payload.Amount.Value)
	if err != nil || !amount.IsPositive() || strings.TrimSpace(payload.ID) == "" || strings.ContainsRune(payload.ID, '\x00') || !payload.Test {
		return Payment{}, errors.New("invalid test payment from YooKassa")
	}
	switch payload.Status {
	case "pending", "waiting_for_capture", "succeeded", "canceled":
	default:
		return Payment{}, errors.New("unknown YooKassa payment status")
	}
	var confirmationURL *string
	if payload.Status == "pending" && payload.Confirmation.URL != nil {
		target, err := url.Parse(*payload.Confirmation.URL)
		if err != nil || target.Scheme != "https" || target.Hostname() == "" || target.User != nil {
			return Payment{}, errors.New("invalid YooKassa confirmation URL")
		}
		confirmationURL = payload.Confirmation.URL
	}
	return Payment{ID: payload.ID, Status: payload.Status, Amount: amount, Currency: payload.Amount.Currency,
		ConfirmationURL: confirmationURL}, nil
}
