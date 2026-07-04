package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/paymentintent"
	"github.com/stripe/stripe-go/v80/webhook"

	"github.com/aetherius/platform/services/payment/internal/model"
	"github.com/aetherius/platform/services/payment/internal/repository"
)

type PaymentService struct {
	repo      *repository.PaymentRepository
	stripeKey string
	whSecret  string
}

func NewPaymentService(repo *repository.PaymentRepository) *PaymentService {
	sk := os.Getenv("STRIPE_SECRET_KEY")
	if sk == "" {
		sk = "sk_test_placeholder"
	}
	wh := os.Getenv("STRIPE_WEBHOOK_SECRET")
	stripe.Key = sk

	return &PaymentService{
		repo:      repo,
		stripeKey: sk,
		whSecret:  wh,
	}
}

func (s *PaymentService) GetWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	return s.repo.GetOrCreateWallet(ctx, userID)
}

func (s *PaymentService) CreatePaymentIntent(ctx context.Context, userID uuid.UUID, req *model.CreatePaymentIntentRequest) (*model.CreatePaymentIntentResponse, error) {
	isDev := s.stripeKey == "sk_test_placeholder"

	wallet, err := s.repo.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	var intentID, clientSecret string
	if isDev {
		intentID = fmt.Sprintf("pi_dev_%s", uuid.New().String()[:8])
		clientSecret = fmt.Sprintf("pi_secret_dev_%s", uuid.New().String())
		go s.ConfirmPayment(context.Background(), intentID)
	} else {
		params := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(req.Amount),
			Currency: stripe.String("usd"),
			Metadata: map[string]string{"user_id": userID.String()},
			AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
				Enabled: stripe.Bool(true),
			},
		}
		intent, err := paymentintent.New(params)
		if err != nil {
			return nil, fmt.Errorf("create payment intent: %w", err)
		}
		intentID = intent.ID
		clientSecret = intent.ClientSecret
	}

	ref := fmt.Sprintf("TOPUP-%s", time.Now().Format("20060102-150405"))
	amountFloat := float64(req.Amount) / 100.0

	txn := &model.Transaction{
		ID:             uuid.New(),
		UserID:         userID,
		WalletID:       wallet.ID,
		Type:           model.TxnDeposit,
		Status:         model.TxnPending,
		Amount:         amountFloat,
		Currency:       "USD",
		Fee:            0,
		NetAmount:      amountFloat,
		BalanceBefore:  wallet.Balance,
		BalanceAfter:   wallet.Balance,
		Description:    req.Description,
		StripeIntentID: &intentID,
		Reference:      ref,
	}

	if err := s.repo.CreateTransaction(ctx, txn); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	return &model.CreatePaymentIntentResponse{
		IntentID:     intentID,
		ClientSecret: clientSecret,
		Amount:       req.Amount,
		Currency:     "usd",
	}, nil
}

func (s *PaymentService) ConfirmPayment(ctx context.Context, intentID string) error {
	txn, err := s.repo.GetTransactionByStripeIntent(ctx, intentID)
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	if txn.Status == model.TxnCompleted {
		return nil
	}

	isDev := s.stripeKey == "sk_test_placeholder"

	var amount float64
	if isDev {
		amount = txn.Amount
	} else {
		intent, err := paymentintent.Get(intentID, nil)
		if err != nil {
			return fmt.Errorf("get payment intent: %w", err)
		}
		if intent.Status != stripe.PaymentIntentStatusSucceeded {
			return nil
		}
		amount = float64(intent.Amount) / 100.0
	}

	if err := s.repo.UpdateBalance(ctx, txn.WalletID, amount); err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	wallet, err := s.repo.GetWalletByUserID(ctx, txn.UserID)
	if err != nil {
		return err
	}

	txn.BalanceBefore = wallet.Balance - amount
	txn.BalanceAfter = wallet.Balance
	txn.Status = model.TxnCompleted
	if err := s.repo.UpdateTransactionStatus(ctx, txn); err != nil {
		return err
	}

	log.Info().Str("user_id", txn.UserID.String()).Float64("amount", amount).Msg("payment confirmed, wallet credited")

	return nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	if s.whSecret == "" {
		return s.handleWebhookLocal(ctx, payload)
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, s.whSecret)
	if err != nil {
		return fmt.Errorf("verify webhook: %w", err)
	}

	switch event.Type {
	case stripe.EventTypePaymentIntentSucceeded:
		var intent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &intent); err != nil {
			return err
		}
		return s.ConfirmPayment(ctx, intent.ID)
	}

	return nil
}

func (s *PaymentService) handleWebhookLocal(ctx context.Context, payload []byte) error {
	var event struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	if event.Type == "payment_intent.succeeded" {
		var data struct {
			Object struct {
				ID string `json:"id"`
			} `json:"object"`
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return err
		}
		return s.ConfirmPayment(ctx, data.Object.ID)
	}
	return nil
}

func (s *PaymentService) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Transaction, error) {
	return s.repo.ListTransactions(ctx, userID, limit, offset)
}

func (s *PaymentService) AddPaymentMethod(ctx context.Context, userID uuid.UUID, stripePMID, last4, brand string, expMonth, expYear int) (*model.PaymentMethod, error) {
	existing, _ := s.repo.ListPaymentMethods(ctx, userID)
	isDefault := len(existing) == 0

	pm := &model.PaymentMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       model.PMCard,
		StripePMID: stripePMID,
		Last4:      last4,
		Brand:      brand,
		ExpMonth:   expMonth,
		ExpYear:    expYear,
		IsDefault:  isDefault,
	}

	if err := s.repo.SavePaymentMethod(ctx, pm); err != nil {
		return nil, err
	}
	return pm, nil
}

func (s *PaymentService) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*model.PaymentMethod, error) {
	return s.repo.ListPaymentMethods(ctx, userID)
}

func (s *PaymentService) DeletePaymentMethod(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeletePaymentMethod(ctx, id, userID)
}

func (s *PaymentService) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	wallet, err := s.repo.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return 0, err
	}
	return wallet.Balance, nil
}

func (s *PaymentService) GetUnpaidInvoiceTotal(ctx context.Context, userID uuid.UUID) (float64, error) {
	return s.repo.GetUnpaidInvoiceAmount(ctx, userID)
}

func (s *PaymentService) PayInvoice(ctx context.Context, userID uuid.UUID, invoiceID uuid.UUID) error {
	invoice, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if invoice.UserID != userID {
		return repository.ErrInvoiceNotFound
	}
	if invoice.Status != "pending" {
		return fmt.Errorf("invoice is not pending")
	}

	wallet, err := s.repo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return err
	}

	newBalance, err := s.repo.DeductBalance(ctx, wallet.ID, invoice.Amount)
	if err != nil {
		return err
	}

	if err := s.repo.MarkInvoicePaid(ctx, invoiceID); err != nil {
		return err
	}

	balanceBefore := wallet.Balance
	txn := &model.Transaction{
		ID:            uuid.New(),
		UserID:        userID,
		WalletID:      wallet.ID,
		Type:          model.TxnPayment,
		Status:        model.TxnCompleted,
		Amount:        invoice.Amount,
		Currency:      "USD",
		Fee:           0,
		NetAmount:     -invoice.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  newBalance,
		Description:   fmt.Sprintf("Payment for invoice %s", invoiceID.String()),
		InvoiceID:     &invoiceID,
		Reference:     fmt.Sprintf("INV-PAY-%s", time.Now().Format("20060102-150405")),
	}

	return s.repo.CreateTransaction(ctx, txn)
}
