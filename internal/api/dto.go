package api

// CreatePurchaseRequest represents a request to create a new purchase
// @Description Purchase creation details
type CreatePurchaseRequest struct {
	Description     string `json:"description" example:"Office supplies purchase"`
	TransactionDate string `json:"transactionDate" example:"2026-06-15"`
	AmountUsd       string `json:"amountUsd" example:"1500.00"`
}

// PurchaseResponse represents a purchase in API responses
// @Description Complete purchase details with optional conversion
type PurchaseResponse struct {
	ID                string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Description       string `json:"description" example:"Office supplies purchase"`
	TransactionDate   string `json:"transactionDate" example:"2026-06-15"`
	AmountUsd         string `json:"amountUsd" example:"1500.00"`
	CreatedAt         string `json:"createdAt" example:"2026-06-18T14:30:00Z"`
	UpdatedAt         string `json:"updatedAt" example:"2026-06-18T14:30:00Z"`
	ConvertedCurrency string `json:"convertedCurrency,omitempty" example:"EUR"`
	RateDate          string `json:"rateDate,omitempty" example:"2026-06-15"`
	Rate              string `json:"rate,omitempty" example:"0.85"`
	ConvertedAmount   string `json:"convertedAmount,omitempty" example:"1275.00"`
}

// HealthResponse represents the health check response
// @Description Health status of the API
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
	Error  string `json:"error,omitempty" example:"database not available"`
}
