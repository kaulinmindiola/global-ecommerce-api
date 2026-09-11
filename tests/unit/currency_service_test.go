package unit_test

import (
	"context"
	"testing"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/fixtures"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/mocks"
)

func TestCurrencyService_Convert(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(m *mocks.MockCurrencyRepository)
		req           service.ConvertRequest
		wantConverted float64
		wantRate      float64
		wantErr       bool
	}{
		{
			name: "USD to EUR conversion",
			setupMock: func(m *mocks.MockCurrencyRepository) {
				m.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
					switch code {
					case "USD":
						return fixtures.USD(), nil
					case "EUR":
						return fixtures.EUR(), nil
					}
					return nil, domain.ErrNotFound
				}
			},
			req: service.ConvertRequest{
				FromCode: "USD",
				ToCode:   "EUR",
				Amount:   100.0,
			},
			// rate = EUR.ExchangeRateToUSD / USD.ExchangeRateToUSD = 0.925 / 1.0 = 0.925
			// converted = 100.0 * 0.925 = 92.5
			wantConverted: 92.5,
			wantRate:      0.925,
			wantErr:       false,
		},
		{
			name: "same currency returns original amount",
			setupMock: func(m *mocks.MockCurrencyRepository) {
				m.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
					return fixtures.USD(), nil
				}
			},
			req: service.ConvertRequest{
				FromCode: "USD",
				ToCode:   "USD",
				Amount:   250.0,
			},
			wantConverted: 250.0,
			wantRate:      1.0,
			wantErr:       false,
		},
		{
			name: "USD to COP conversion",
			setupMock: func(m *mocks.MockCurrencyRepository) {
				m.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
					switch code {
					case "USD":
						return fixtures.USD(), nil
					case "COP":
						return fixtures.COP(), nil
					}
					return nil, domain.ErrNotFound
				}
			},
			req: service.ConvertRequest{
				FromCode: "USD",
				ToCode:   "COP",
				Amount:   1.0,
			},
			// rate = 3950 / 1.0 = 3950.0
			wantConverted: 3950.0,
			wantRate:      3950.0,
			wantErr:       false,
		},
		{
			name: "unknown currency returns error",
			setupMock: func(m *mocks.MockCurrencyRepository) {
				m.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
					return nil, domain.ErrNotFound
				}
			},
			req: service.ConvertRequest{
				FromCode: "XYZ",
				ToCode:   "USD",
				Amount:   100.0,
			},
			wantErr: true,
		},
		{
			name:      "zero amount returns error",
			setupMock: func(_ *mocks.MockCurrencyRepository) {},
			req: service.ConvertRequest{
				FromCode: "USD",
				ToCode:   "EUR",
				Amount:   0,
			},
			wantErr: true,
		},
		{
			name:      "negative amount returns error",
			setupMock: func(_ *mocks.MockCurrencyRepository) {},
			req: service.ConvertRequest{
				FromCode: "USD",
				ToCode:   "EUR",
				Amount:   -50,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockCurrencyRepository()
			tt.setupMock(mockRepo)

			svc := service.NewCurrencyService(mockRepo, mocks.NewMockCacheRepository())
			result, err := svc.Convert(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ConvertedAmount != tt.wantConverted {
				t.Errorf("converted amount: got %.4f, want %.4f",
					result.ConvertedAmount, tt.wantConverted)
			}

			if result.ExchangeRate != tt.wantRate {
				t.Errorf("exchange rate: got %.6f, want %.6f",
					result.ExchangeRate, tt.wantRate)
			}

			if result.FromCurrency != tt.req.FromCode {
				t.Errorf("from_currency: got %s, want %s", result.FromCurrency, tt.req.FromCode)
			}

			if result.ToCurrency != tt.req.ToCode {
				t.Errorf("to_currency: got %s, want %s", result.ToCurrency, tt.req.ToCode)
			}
		})
	}
}

func TestCurrencyService_GetExchangeRate(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		wantRate float64
		wantErr  bool
	}{
		{
			name:     "USD to USD is 1.0",
			from:     "USD",
			to:       "USD",
			wantRate: 1.0,
		},
		{
			name:     "USD to EUR",
			from:     "USD",
			to:       "EUR",
			wantRate: 0.925,
		},
		{
			name:     "EUR to USD (inverse)",
			from:     "EUR",
			to:       "USD",
			wantRate: 1.081081, // 1 / 0.925 ≈ 1.081081
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockCurrencyRepository()
			mockRepo.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
				switch code {
				case "USD":
					return fixtures.USD(), nil
				case "EUR":
					return fixtures.EUR(), nil
				}
				return nil, domain.ErrNotFound
			}

			svc := service.NewCurrencyService(mockRepo, mocks.NewMockCacheRepository())
			rate, err := svc.GetExchangeRate(context.Background(), tt.from, tt.to)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Allow a small floating-point tolerance.
			tolerance := 0.000001
			diff := rate - tt.wantRate
			if diff < -tolerance || diff > tolerance {
				t.Errorf("exchange rate: got %.6f, want %.6f (tolerance %.6f)",
					rate, tt.wantRate, tolerance)
			}
		})
	}
}
