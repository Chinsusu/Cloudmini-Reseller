package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// HTTPBillingClient implements BillingClient by calling billing-service REST API.
type HTTPBillingClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPBillingClient creates an HTTPBillingClient.
func NewHTTPBillingClient(baseURL string) *HTTPBillingClient {
	return &HTTPBillingClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10e9}, // 10s
	}
}

func (c *HTTPBillingClient) Hold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID) error {
	body := map[string]any{"user_id": userID, "amount": amount, "reference_type": refType, "reference_id": refID}
	return c.post(ctx, "/internal/billing/hold", body)
}

func (c *HTTPBillingClient) ConfirmHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID, description string) error {
	body := map[string]any{"user_id": userID, "amount": amount, "reference_type": refType, "reference_id": refID, "description": description}
	return c.post(ctx, "/internal/billing/confirm-hold", body)
}

func (c *HTTPBillingClient) ReleaseHold(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, refType string, refID uuid.UUID, description string) error {
	body := map[string]any{"user_id": userID, "amount": amount, "reference_type": refType, "reference_id": refID, "description": description}
	return c.post(ctx, "/internal/billing/release-hold", body)
}

func (c *HTTPBillingClient) CalculatePrice(ctx context.Context, baseCost decimal.Decimal, productType string, productID uuid.UUID, resellerID *uuid.UUID) (decimal.Decimal, error) {
	body := map[string]any{"base_cost": baseCost, "product_type": productType, "product_id": productID}
	if resellerID != nil {
		body["reseller_id"] = resellerID
	}

	respBody, err := c.postWithResponse(ctx, "/internal/billing/calculate-price", body)
	if err != nil {
		return decimal.Zero, err
	}

	var resp struct {
		Data struct {
			Price string `json:"price"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return decimal.Zero, fmt.Errorf("HTTPBillingClient.CalculatePrice: parse response: %w", err)
	}

	price, err := decimal.NewFromString(resp.Data.Price)
	if err != nil {
		return decimal.Zero, fmt.Errorf("HTTPBillingClient.CalculatePrice: parse price: %w", err)
	}
	return price, nil
}

func (c *HTTPBillingClient) post(ctx context.Context, path string, body any) error {
	_, err := c.postWithResponse(ctx, path, body)
	return err
}

func (c *HTTPBillingClient) postWithResponse(ctx context.Context, path string, body any) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("HTTPBillingClient: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("HTTPBillingClient: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "proxy-service")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTPBillingClient %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 402 {
		return nil, fmt.Errorf("insufficient funds")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTPBillingClient %s: status %d", path, resp.StatusCode)
	}

	// Use io.ReadAll to read the complete response body (not fmt.Fscan which truncates at whitespace)
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTTPBillingClient %s: read body: %w", path, err)
	}
	return buf, nil
}
