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

// ListCurrencies godoc
// @Summary      List all active currencies
// @Description  Get a list of all active supported currencies for the platform
// @Tags         Currencies
// @Produce      json
// @Success      200  {object}  map[string]any "data: array of currencies, total: integer"
// @Failure      500  {object}  ErrorResponse  "Internal Server Error"
// @Router       /currencies [get]
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

// ConvertCurrency godoc
// @Summary      Convert currency amount
// @Description  Performs a real-time currency conversion using the latest exchange rates
// @Tags         Currencies
// @Produce      json
// @Param        from   query     string  true  "Source ISO 4217 currency code (e.g., USD)"
// @Param        to     query     string  true  "Target ISO 4217 currency code (e.g., EUR)"
// @Param        amount query     number  true  "Amount to convert (must be a positive number)"
// @Success      200    {object}  any            "Successful conversion result"
// @Failure      400    {object}  ErrorResponse  "Validation error (missing or invalid parameters)"
// @Failure      404    {object}  ErrorResponse  "Currency code not found"
// @Failure      500    {object}  ErrorResponse  "Internal Server Error"
// @Router       /currencies/convert [get]
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
