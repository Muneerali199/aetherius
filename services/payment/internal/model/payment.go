package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TxnDeposit  TransactionType = "deposit"
	TxnWithdraw TransactionType = "withdraw"
	TxnPayment  TransactionType = "payment"
	TxnRefund   TransactionType = "refund"
	TxnCredit   TransactionType = "credit"
	TxnFee      TransactionType = "fee"
)

type TransactionStatus string

const (
	TxnPending   TransactionStatus = "pending"
	TxnCompleted TransactionStatus = "completed"
	TxnFailed    TransactionStatus = "failed"
	TxnCancelled TransactionStatus = "cancelled"
)

type PaymentMethodType string

const (
	PMCard   PaymentMethodType = "card"
	PMCrypto PaymentMethodType = "crypto"
	PMBank   PaymentMethodType = "bank_transfer"
)

type Wallet struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Balance   float64   `json:"balance" db:"balance"`
	Currency  string    `json:"currency" db:"currency"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Transaction struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	UserID         uuid.UUID         `json:"user_id" db:"user_id"`
	WalletID       uuid.UUID         `json:"wallet_id" db:"wallet_id"`
	Type           TransactionType   `json:"type" db:"txn_type"`
	Status         TransactionStatus `json:"status" db:"status"`
	Amount         float64           `json:"amount" db:"amount"`
	Currency       string            `json:"currency" db:"currency"`
	Fee            float64           `json:"fee,omitempty" db:"fee"`
	NetAmount      float64           `json:"net_amount" db:"net_amount"`
	BalanceBefore  float64           `json:"balance_before" db:"balance_before"`
	BalanceAfter   float64           `json:"balance_after" db:"balance_after"`
	Description    string            `json:"description" db:"description"`
	StripeIntentID *string           `json:"stripe_intent_id,omitempty" db:"stripe_intent_id"`
	InvoiceID      *uuid.UUID        `json:"invoice_id,omitempty" db:"invoice_id"`
	Reference      string            `json:"reference,omitempty" db:"reference"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
}

type PaymentMethod struct {
	ID         uuid.UUID         `json:"id" db:"id"`
	UserID     uuid.UUID         `json:"user_id" db:"user_id"`
	Type       PaymentMethodType `json:"type" db:"pm_type"`
	StripePMID string            `json:"stripe_pm_id" db:"stripe_pm_id"`
	Last4      string            `json:"last4,omitempty" db:"last4"`
	Brand      string            `json:"brand,omitempty" db:"brand"`
	ExpMonth   int               `json:"exp_month,omitempty" db:"exp_month"`
	ExpYear    int               `json:"exp_year,omitempty" db:"exp_year"`
	IsDefault  bool              `json:"is_default" db:"is_default"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
}

type CreatePaymentIntentRequest struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

type CreatePaymentIntentResponse struct {
	IntentID     string `json:"intent_id"`
	ClientSecret string `json:"client_secret"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
}

type TopUpRequest struct {
	Amount          int64  `json:"amount"`
	PaymentMethodID string `json:"payment_method_id,omitempty"`
}

type WithdrawRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type Invoice struct {
	ID     uuid.UUID `json:"id" db:"id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`
	Amount float64   `json:"amount" db:"amount"`
	Status string    `json:"status" db:"status"`
}

type PaymentWebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID           string            `json:"id"`
			Amount       int64             `json:"amount"`
			Currency     string            `json:"currency"`
			Status       string            `json:"status"`
			ClientSecret string            `json:"client_secret"`
			Metadata     map[string]string `json:"metadata"`
		} `json:"object"`
	} `json:"data"`
}
