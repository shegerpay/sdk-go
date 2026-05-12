// ShegerPay Go SDK v2.2.0
// Official Go SDK for ShegerPay Payment Verification Gateway
//
// Usage:
//   client := shegerpay.NewClient("sk_test_xxx")
//   result, err := client.Verify(shegerpay.VerifyParams{
//       TransactionID: "FT123456",
//       Amount: 100,
//   })

package shegerpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	Version        = "1.0.0"
	DefaultBaseURL = "https://api.shegerpay.com"
)

// Errors
var (
	ErrInvalidAPIKey = errors.New("invalid API key format")
	ErrMissingAPIKey = errors.New("API key is required")
)

// VerificationResult represents the result of a payment verification
type VerificationResult struct {
	Verified      bool    `json:"verified"`
	Valid         bool    `json:"valid"`
	Status        string  `json:"status"`
	Provider      string  `json:"provider,omitempty"`
	TransactionID string  `json:"transaction_id,omitempty"`
	Amount        float64 `json:"amount,omitempty"`
	Reason        string  `json:"reason,omitempty"`
	Mode          string  `json:"mode,omitempty"`
}

// VerifyParams contains parameters for verification
type VerifyParams struct {
	Provider      string
	TransactionID string
	Amount        float64
	MerchantName  string
	SubProvider   string
	SenderAccount string
}

// Client is the ShegerPay API client
type Client struct {
	apiKey  string
	baseURL string
	mode    string
	http    *http.Client
}

// NewClient creates a new ShegerPay client
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	
	if !strings.HasPrefix(apiKey, "sk_test_") && !strings.HasPrefix(apiKey, "sk_live_") {
		return nil, ErrInvalidAPIKey
	}
	
	mode := "live"
	if strings.HasPrefix(apiKey, "sk_test_") {
		mode = "test"
	}
	
	client := &Client{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		mode:    mode,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	
	for _, opt := range opts {
		opt(client)
	}
	
	return client, nil
}

// ClientOption is a function that configures the client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithTimeout sets request timeout
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

// Verify verifies a payment transaction
func (c *Client) Verify(params VerifyParams) (*VerificationResult, error) {
	if params.TransactionID == "" {
		return nil, errors.New("TransactionID is required")
	}
	if params.Amount <= 0 {
		return nil, errors.New("Amount is required")
	}
	
	provider := params.Provider
	if provider == "" {
		if strings.Contains(strings.ToLower(params.TransactionID), "cs.bankofabyssinia.com/slip/?trx=") {
			provider = "boa"
		}
	}
	if provider == "" {
		return nil, errors.New("Provider is required for ambiguous transaction references. Pass Provider explicitly or use QuickVerify")
	}
	
	merchantName := params.MerchantName
	if merchantName == "" {
		merchantName = "ShegerPay Verification"
	}
	
	data := url.Values{}
	data.Set("provider", provider)
	data.Set("transaction_id", params.TransactionID)
	data.Set("amount", fmt.Sprintf("%f", params.Amount))
	data.Set("merchant_name", merchantName)
	
	if params.SubProvider != "" {
		data.Set("sub_provider", params.SubProvider)
	}
	if params.SenderAccount != "" {
		data.Set("sender_account", params.SenderAccount)
	}
	
	result := &VerificationResult{}
	err := c.request("POST", "/api/v1/verify", data, result)
	return result, err
}

// QuickVerify verifies with auto-detected provider
func (c *Client) QuickVerify(transactionID string, amount float64) (*VerificationResult, error) {
	data := url.Values{}
	data.Set("transaction_id", transactionID)
	data.Set("amount", fmt.Sprintf("%f", amount))
	
	result := &VerificationResult{}
	err := c.request("POST", "/api/v1/quick-verify", data, result)
	return result, err
}

// GetHistory gets transaction history
func (c *Client) GetHistory() ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := c.request("GET", "/api/v1/history", nil, &result)
	return result, err
}

func (c *Client) request(method, path string, data url.Values, result interface{}) error {
	fullURL := c.baseURL + path
	
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}
	
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return err
	}
	
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", "ShegerPay-Go-SDK/1.0")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	if resp.StatusCode == 401 {
		return errors.New("invalid API key")
	}
	if resp.StatusCode == 400 {
		var errResp map[string]string
		json.Unmarshal(respBody, &errResp)
		return errors.New(errResp["detail"])
	}
	
	return json.Unmarshal(respBody, result)
}

// VerifyImage verifies a payment using a receipt screenshot (base64 or URL)
func (c *Client) VerifyImage(imageData string, opts ...map[string]string) (*VerificationResult, error) {
	params := url.Values{}
	params.Set("image", imageData)
	if len(opts) > 0 {
		for k, v := range opts[0] {
			params.Set(k, v)
		}
	}
	var result VerificationResult
	if err := c.request("POST", "/api/v1/verify/image", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePaymentLink creates a shareable payment link
func (c *Client) CreatePaymentLink(title string, amount float64, opts ...map[string]string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("title", title)
	params.Set("amount", fmt.Sprintf("%.2f", amount))
	params.Set("currency", "ETB")
	if len(opts) > 0 {
		for k, v := range opts[0] {
			params.Set(k, v)
		}
	}
	var result map[string]interface{}
	if err := c.request("POST", "/api/v1/payment-links", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListPaymentLinks returns all payment links for the account
func (c *Client) ListPaymentLinks() ([]map[string]interface{}, error) {
	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := c.request("GET", "/api/v1/payment-links", nil, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// DeletePaymentLink deletes a payment link by ID
func (c *Client) DeletePaymentLink(linkID string) error {
	var result map[string]interface{}
	return c.request("DELETE", "/api/v1/payment-links/"+linkID, nil, &result)
}

// VerifyWebhookSignature verifies a webhook signature
func VerifyWebhookSignature(payload, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
