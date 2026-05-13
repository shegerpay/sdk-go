
// ============================================
// WALLET METHODS
// ============================================

// GetWalletBalance gets PayPal wallet balance.
func (c *Client) GetWalletBalance() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.request("GET", "/api/v1/paypal/wallet/balance", nil, &result)
	return result, err
}

// ConvertCurrency is private/assisted and not exposed in the public SDK.
func (c *Client) ConvertCurrency(from, to string, amount float64) (map[string]interface{}, error) {
	return nil, fmt.Errorf("currency conversion is private/assisted and is not exposed in the public SDK")
}

// ============================================
// REFUND METHODS
// ============================================

// CreateRefund requests a refund
func (c *Client) CreateRefund(transactionID string, amount float64, reason string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("transaction_id", transactionID)
	if amount > 0 {
		data.Set("amount", fmt.Sprintf("%f", amount))
	}
	if reason != "" {
		data.Set("reason", reason)
	}
	
	var result map[string]interface{}
	err := c.request("POST", "/api/v1/refunds/request", data, &result)
	return result, err
}

// ApproveRefund approves a pending refund
func (c *Client) ApproveRefund(refundID string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.request("POST", fmt.Sprintf("/api/v1/refunds/%s/approve", refundID), nil, &result)
	return result, err
}

// ============================================
// DISPUTE METHODS
// ============================================

// ListDisputes lists disputes
func (c *Client) ListDisputes(status string) ([]map[string]interface{}, error) {
	path := "/api/v1/disputes"
	if status != "" {
		path += "?status=" + status
	}
	
	var result []map[string]interface{}
	err := c.request("GET", path, nil, &result)
	return result, err
}

// RespondToDispute responds to a dispute
func (c *Client) RespondToDispute(disputeID, message string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("message", message)
	
	var result map[string]interface{}
	err := c.request("POST", fmt.Sprintf("/api/v1/disputes/%s/respond", disputeID), data, &result)
	return result, err
}

// ============================================
// ANALYTICS METHODS
// ============================================

// GetAPIUsage gets usage stats
func (c *Client) GetAPIUsage() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.request("GET", "/api/v1/analytics/api-usage", nil, &result)
	return result, err
}
