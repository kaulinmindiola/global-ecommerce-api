package handler

import (
	"net/http"
	"strconv"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// CurrencyHandler handles all currency HTTP endpoints.
// GET /api/v1/currencies
// GET /api/v1/currencies/convert
type CurrencyHandler struct {
	currencyService service.CurrencyService
}

// NewCurrencyHandler creates a new CurrencyHandler.
func NewCurrencyHandler(currencyService service.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{currencyService: currencyService}
}

// ─────────────────────────────────────────────
// GET /api/v1/currencies
// ─────────────────────────────────────────────

// ListCurrencies returns all active supported currencies.
// Response: 200 OK — matches spec page 8
func (h *CurrencyHandler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.currencyService.ListActive(r.Context())
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, map[string]any{
		"data":  currencies,
		"total": len(currencies),
	})
}

// ─────────────────────────────────────────────
// GET /api/v1/currencies/convert
// ─────────────────────────────────────────────

// ConvertCurrency performs a real-time currency conversion.
// Query params: from, to, amount
// Example: /api/v1/currencies/convert?from=USD&to=EUR&amount=100
// Response: 200 OK — matches spec page 10
func (h *CurrencyHandler) ConvertCurrency(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	fromCode := q.Get("from")
	toCode := q.Get("to")
	amountStr := q.Get("amount")

	// Validate all three query parameters are present.
	if fromCode == "" || toCode == "" || amountStr == "" {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Query parameters 'from', 'to', and 'amount' are required",
			ErrorDetail{Field: "from", Message: "Source currency code is required", Code: "REQUIRED"},
			ErrorDetail{Field: "to", Message: "Target currency code is required", Code: "REQUIRED"},
			ErrorDetail{Field: "amount", Message: "Amount to convert is required", Code: "REQUIRED"},
		)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Amount must be a positive number",
			ErrorDetail{Field: "amount", Message: "Must be a positive numeric value", Code: "INVALID_FORMAT"},
		)
		return
	}

	result, err := h.currencyService.Convert(r.Context(), service.ConvertRequest{
		FromCode: fromCode,
		ToCode:   toCode,
		Amount:   amount,
	})
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, result)
}
