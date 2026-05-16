package duitku

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client handles Duitku API communication
type Client struct {
	merchantCode string
	apiKey       string
	baseURL      string
	callbackURL  string
	returnURL    string
	httpClient   *http.Client
	logger       *zap.Logger
}

// Config holds Duitku configuration
type Config struct {
	MerchantCode string
	APIKey       string
	BaseURL      string // https://sandbox.duitku.com or https://passport.duitku.com
	CallbackURL  string
	ReturnURL    string
}

// InquiryRequest represents a payment inquiry request
type InquiryRequest struct {
	MerchantCode    string `json:"merchantCode"`
	PaymentAmount   int64  `json:"paymentAmount"`
	MerchantOrderID string `json:"merchantOrderId"`
	ProductDetails  string `json:"productDetails"`
	Email           string `json:"email"`
	PaymentMethod   string `json:"paymentMethod"` // VC, BK, M1, etc.
	CustomerVAName  string `json:"customerVaName"`
	CallbackURL     string `json:"callbackUrl"`
	ReturnURL       string `json:"returnUrl"`
	Signature       string `json:"signature"`
	ExpiryPeriod    int    `json:"expiryPeriod"` // in minutes
}

// InquiryResponse from Duitku API
type InquiryResponse struct {
	MerchantCode    string `json:"merchantCode"`
	Reference       string `json:"reference"`
	PaymentURL      string `json:"paymentUrl"`
	VANumber        string `json:"vaNumber"`
	Amount          string `json:"amount"`
	MerchantOrderID string `json:"merchantOrderId"`
	StatusCode      string `json:"statusCode"`
	StatusMessage   string `json:"statusMessage"`
}

// CallbackPayload represents the callback from Duitku
type CallbackPayload struct {
	MerchantCode      string `json:"merchantCode"`
	Amount            string `json:"amount"`
	MerchantOrderID   string `json:"merchantOrderId"`
	ProductDetail     string `json:"productDetail"`
	ResultCode        string `json:"resultCode"` // 00 = success, 01 = pending
	Signature         string `json:"signature"`
	Reference         string `json:"reference"`
	PaymentCode       string `json:"paymentCode"`
	PublisherOrderID  string `json:"publisherOrderId"`
}

// NewClient creates a new Duitku API client
func NewClient(cfg *Config, logger *zap.Logger) *Client {
	return &Client{
		merchantCode: cfg.MerchantCode,
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		callbackURL:  cfg.CallbackURL,
		returnURL:    cfg.ReturnURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// CreateInquiry initiates a payment request to Duitku
func (c *Client) CreateInquiry(ctx context.Context, merchantOrderID string, amount int64, productDetails, email, paymentMethod, customerName string) (*InquiryResponse, error) {
	// Generate signature: MD5(merchantCode + merchantOrderId + paymentAmount + apiKey)
	sigStr := fmt.Sprintf("%s%s%d%s", c.merchantCode, merchantOrderID, amount, c.apiKey)
	hash := md5.Sum([]byte(sigStr))
	signature := hex.EncodeToString(hash[:])

	req := InquiryRequest{
		MerchantCode:    c.merchantCode,
		PaymentAmount:   amount,
		MerchantOrderID: merchantOrderID,
		ProductDetails:  productDetails,
		Email:           email,
		PaymentMethod:   paymentMethod,
		CustomerVAName:  customerName,
		CallbackURL:     c.callbackURL,
		ReturnURL:       c.returnURL,
		Signature:       signature,
		ExpiryPeriod:    1440, // 24 hours
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal inquiry request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/webapi/api/merchant/v2/inquiry", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("duitku api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Duitku API error", zap.Int("status", resp.StatusCode), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("duitku api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result InquiryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal inquiry response: %w", err)
	}

	if result.StatusCode != "00" {
		return nil, fmt.Errorf("duitku inquiry failed: %s", result.StatusMessage)
	}

	return &result, nil
}

// VerifyCallbackSignature verifies the HMAC SHA-256 signature from Duitku callback
func (c *Client) VerifyCallbackSignature(payload *CallbackPayload) bool {
	// Duitku callback signature: MD5(merchantCode + amount + merchantOrderId + apiKey)
	sigStr := fmt.Sprintf("%s%s%s%s", c.merchantCode, payload.Amount, payload.MerchantOrderID, c.apiKey)
	hash := md5.Sum([]byte(sigStr))
	expected := hex.EncodeToString(hash[:])
	return hmac.Equal([]byte(expected), []byte(payload.Signature))
}

// GenerateSignature generates HMAC SHA-256 for outgoing requests
func (c *Client) GenerateSignature(data string) string {
	mac := hmac.New(sha256.New, []byte(c.apiKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
