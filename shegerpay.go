// ShegerPay Go SDK v2.2.1
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
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	Version        = "2.2.1"
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

// QuickVerify verifies with auto-detected provider. transactionID may be a typed
// reference OR a raw scanned-QR payload — it is decoded server-side.
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
	req.Header.Set("User-Agent", "ShegerPay-Go-SDK/"+Version)
	if data != nil {
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
	if resp.StatusCode >= 400 {
		return fmt.Errorf("shegerpay request failed: %s", string(respBody))
	}
	if resp.StatusCode == http.StatusNoContent || result == nil {
		return nil
	}
	
	return json.Unmarshal(respBody, result)
}

func (c *Client) requestJSON(method, path string, payload map[string]interface{}, result interface{}) error {
	fullURL := c.baseURL + path
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", "ShegerPay-Go-SDK/"+Version)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNoContent || result == nil {
		return nil
	}
	if resp.StatusCode == 401 {
		return errors.New("invalid API key")
	}
	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.Unmarshal(respBody, &errResp)
		if detail, ok := errResp["detail"].(string); ok && detail != "" {
			return errors.New(detail)
		}
		return fmt.Errorf("shegerpay request failed: %s", string(respBody))
	}
	return json.Unmarshal(respBody, result)
}

// VerifyImage verifies a payment from a receipt image/screenshot (or PDF).
//
// Works for ANY supported bank — the backend reads the receipt's QR code (CBE,
// Telebirr, BOA…) or OCRs the reference and auto-detects the provider. Just pass
// the raw image bytes; no need to know the bank or pre-extract the reference.
// Optional fields (amount, provider, transaction_id, merchant_name,
// sender_account) may be passed via opts.
func (c *Client) VerifyImage(screenshot []byte, opts ...map[string]string) (*VerificationResult, error) {
	fields := map[string]string{}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			fields[k] = v
		}
	}
	var result VerificationResult
	if err := c.requestMultipart("/api/v1/verify-image", fields, "screenshot", "receipt.png", screenshot, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// requestMultipart POSTs multipart/form-data with a single file part plus any
// extra form fields. Used by VerifyImage (the endpoint requires the file part
// to be named exactly "screenshot").
func (c *Client) requestMultipart(path string, fields map[string]string, fileField, fileName string, fileData []byte, result interface{}) error {
	fullURL := c.baseURL + path

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return err
		}
	}
	fw, err := w.CreateFormFile(fileField, fileName)
	if err != nil {
		return err
	}
	if _, err := fw.Write(fileData); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fullURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", "ShegerPay-Go-SDK/"+Version)
	req.Header.Set("Content-Type", w.FormDataContentType())

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
	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.Unmarshal(respBody, &errResp)
		if detail, ok := errResp["detail"].(string); ok && detail != "" {
			return errors.New(detail)
		}
		return fmt.Errorf("shegerpay request failed: %s", string(respBody))
	}
	if resp.StatusCode == http.StatusNoContent || result == nil {
		return nil
	}
	return json.Unmarshal(respBody, result)
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
		Links []map[string]interface{} `json:"links"`
	}
	if err := c.request("GET", "/api/v1/payment-links", nil, &result); err != nil {
		return nil, err
	}
	return result.Links, nil
}

// DeletePaymentLink deletes a payment link by ID
func (c *Client) DeletePaymentLink(linkID string) error {
	var result map[string]interface{}
	return c.request("DELETE", "/api/v1/payment-links/"+linkID, nil, &result)
}

// CreatePromoCode creates a reusable ShegerPay promo code. Requires a secret key and discount_codes entitlement.
func (c *Client) CreatePromoCode(params map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.requestJSON("POST", "/api/v1/promo-codes/", promoPayload(params), &result)
	return result, err
}

// ListPromoCodes lists reusable promo codes for the merchant.
func (c *Client) ListPromoCodes() ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := c.request("GET", "/api/v1/promo-codes/", nil, &result)
	return result, err
}

// UpdatePromoCode updates a reusable promo code.
func (c *Client) UpdatePromoCode(codeID string, params map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.requestJSON("PATCH", "/api/v1/promo-codes/"+codeID, promoPayload(params), &result)
	return result, err
}

// DeletePromoCode deletes a reusable promo code.
func (c *Client) DeletePromoCode(codeID string) error {
	return c.request("DELETE", "/api/v1/promo-codes/"+codeID, nil, nil)
}

// ValidatePromoCode previews a promo code before payment. It does not consume usage.
func (c *Client) ValidatePromoCode(code string, amount float64, opts map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{"code": code, "amount": amount}
	for k, v := range opts {
		payload[k] = v
	}
	var result map[string]interface{}
	err := c.requestJSON("POST", "/api/v1/promo-codes/validate", payload, &result)
	return result, err
}

// RedeemPromoCode consumes a promo code once after a verified transaction.
func (c *Client) RedeemPromoCode(code string, amount float64, transactionID string, opts map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{"code": code, "amount": amount, "transaction_id": transactionID}
	for k, v := range opts {
		payload[k] = v
	}
	var result map[string]interface{}
	err := c.requestJSON("POST", "/api/v1/promo-codes/redeem", payload, &result)
	return result, err
}

// ApplyPaymentLinkCoupon previews a promo code against a ShegerPay payment link.
func (c *Client) ApplyPaymentLinkCoupon(shortCode, code string, amount float64, quantity int, opts ...map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{"code": code, "quantity": quantity}
	if amount > 0 {
		payload["amount"] = amount
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			payload[k] = v
		}
	}
	var result map[string]interface{}
	err := c.requestJSON("POST", "/api/v1/payment-links/"+shortCode+"/apply-coupon", payload, &result)
	return result, err
}

// GetPaymentLinkOrderStatus returns the source-of-truth status for one checkout order.
func (c *Client) GetPaymentLinkOrderStatus(shortCode, orderID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.request("GET", "/api/v1/payment-links/"+shortCode+"/orders/"+orderID+"/status", nil, &result)
	return result, err
}

func promoPayload(params map[string]interface{}) map[string]interface{} {
	payload := map[string]interface{}{}
	for k, v := range params {
		switch k {
		case "discountType":
			payload["discount_type"] = v
		case "discountValue":
			payload["discount_value"] = v
		case "discountPercent":
			payload["discount_percent"] = v
		case "maxDiscountAmount":
			payload["max_discount_amount"] = v
		case "minOrderAmount":
			payload["min_order_amount"] = v
		case "maxUses":
			payload["max_uses"] = v
		case "maxUsesPerCustomer":
			payload["max_uses_per_customer"] = v
		case "startsAt":
			payload["starts_at"] = v
		case "expiresAt":
			payload["expires_at"] = v
		case "appliesToLinkIds":
			payload["applies_to_link_ids"] = v
		case "allowedProviders":
			payload["allowed_providers"] = v
		default:
			payload[k] = v
		}
	}
	return payload
}

// VerifyWebhookSignature verifies a webhook signature
func VerifyWebhookSignature(payload, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyRedirectSignature verifies signed payment-link redirect parameters.
func VerifyRedirectSignature(params map[string]interface{}, signature, secret string) bool {
	amount, _ := strconv.ParseFloat(fmt.Sprint(params["amount"]), 64)
	payload := strings.Join([]string{
		fmt.Sprint(firstRedirectParam(params, "checkout_session_id", "checkoutSessionId")),
		fmt.Sprint(firstRedirectParam(params, "order_id", "orderId")),
		fmt.Sprint(firstRedirectParam(params, "short_code", "shortCode")),
		fmt.Sprintf("%.2f", amount),
		redirectParamDefault(params, "currency", "ETB"),
		redirectParamDefault(params, "status", "paid"),
	}, "|")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimPrefix(signature, "sha256=")))
}

func firstRedirectParam(params map[string]interface{}, snake, camel string) interface{} {
	if value, ok := params[snake]; ok && value != nil {
		return value
	}
	return params[camel]
}

func redirectParamDefault(params map[string]interface{}, key, fallback string) string {
	if value, ok := params[key]; ok && value != nil && fmt.Sprint(value) != "" {
		return fmt.Sprint(value)
	}
	return fallback
}
