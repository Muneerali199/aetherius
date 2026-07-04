package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/payment/internal/model"
)

var (
	ErrWalletNotFound        = errors.New("wallet not found")
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrPaymentMethodNotFound = errors.New("payment method not found")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrInvoiceNotFound       = errors.New("invoice not found")
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

func (r *PaymentRepository) GetOrCreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	wallet, err := r.GetWalletByUserID(ctx, userID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, ErrWalletNotFound) {
		return nil, err
	}
	query := `INSERT INTO wallets (id, user_id, balance, currency) VALUES ($1, $2, 0, 'USD') RETURNING id, user_id, balance, currency, created_at, updated_at`
	wallet = &model.Wallet{}
	err = r.pool.QueryRow(ctx, query, uuid.New(), userID).Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.Currency, &wallet.CreatedAt, &wallet.UpdatedAt)
	return wallet, err
}

func (r *PaymentRepository) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	query := `SELECT id, user_id, balance, currency, created_at, updated_at FROM wallets WHERE user_id = $1`
	w := &model.Wallet{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.Currency, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWalletNotFound
	}
	return w, err
}

func (r *PaymentRepository) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount float64) error {
	query := `UPDATE wallets SET balance = balance + $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, amount, walletID)
	return err
}

func (r *PaymentRepository) CreateTransaction(ctx context.Context, txn *model.Transaction) error {
	query := `INSERT INTO transactions (id, user_id, wallet_id, txn_type, status, amount, currency, fee, net_amount, balance_before, balance_after, description, stripe_intent_id, invoice_id, reference, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW()) RETURNING created_at`
	return r.pool.QueryRow(ctx, query, txn.ID, txn.UserID, txn.WalletID, txn.Type, txn.Status, txn.Amount, txn.Currency, txn.Fee, txn.NetAmount, txn.BalanceBefore, txn.BalanceAfter, txn.Description, txn.StripeIntentID, txn.InvoiceID, txn.Reference).Scan(&txn.CreatedAt)
}

func (r *PaymentRepository) UpdateTransactionStatus(ctx context.Context, txn *model.Transaction) error {
	_, err := r.pool.Exec(ctx, `UPDATE transactions SET status = $1, balance_before = $2, balance_after = $3 WHERE id = $4`,
		txn.Status, txn.BalanceBefore, txn.BalanceAfter, txn.ID)
	return err
}

func (r *PaymentRepository) GetTransactionByStripeIntent(ctx context.Context, intentID string) (*model.Transaction, error) {
	query := `SELECT id, user_id, wallet_id, txn_type, status, amount, currency, fee, net_amount, balance_before, balance_after, description, stripe_intent_id, invoice_id, reference, created_at FROM transactions WHERE stripe_intent_id = $1`
	t := &model.Transaction{}
	err := r.pool.QueryRow(ctx, query, intentID).Scan(&t.ID, &t.UserID, &t.WalletID, &t.Type, &t.Status, &t.Amount, &t.Currency, &t.Fee, &t.NetAmount, &t.BalanceBefore, &t.BalanceAfter, &t.Description, &t.StripeIntentID, &t.InvoiceID, &t.Reference, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTransactionNotFound
	}
	return t, err
}

func (r *PaymentRepository) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Transaction, error) {
	query := `SELECT id, user_id, wallet_id, txn_type, status, amount, currency, fee, net_amount, balance_before, balance_after, description, stripe_intent_id, invoice_id, reference, created_at FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txns []*model.Transaction
	for rows.Next() {
		t := &model.Transaction{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.WalletID, &t.Type, &t.Status, &t.Amount, &t.Currency, &t.Fee, &t.NetAmount, &t.BalanceBefore, &t.BalanceAfter, &t.Description, &t.StripeIntentID, &t.InvoiceID, &t.Reference, &t.CreatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *PaymentRepository) SavePaymentMethod(ctx context.Context, pm *model.PaymentMethod) error {
	query := `INSERT INTO payment_methods (id, user_id, pm_type, stripe_pm_id, last4, brand, exp_month, exp_year, is_default) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING created_at`
	return r.pool.QueryRow(ctx, query, pm.ID, pm.UserID, pm.Type, pm.StripePMID, pm.Last4, pm.Brand, pm.ExpMonth, pm.ExpYear, pm.IsDefault).Scan(&pm.CreatedAt)
}

func (r *PaymentRepository) ListPaymentMethods(ctx context.Context, userID uuid.UUID) ([]*model.PaymentMethod, error) {
	query := `SELECT id, user_id, pm_type, stripe_pm_id, last4, brand, exp_month, exp_year, is_default, created_at FROM payment_methods WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pms []*model.PaymentMethod
	for rows.Next() {
		pm := &model.PaymentMethod{}
		if err := rows.Scan(&pm.ID, &pm.UserID, &pm.Type, &pm.StripePMID, &pm.Last4, &pm.Brand, &pm.ExpMonth, &pm.ExpYear, &pm.IsDefault, &pm.CreatedAt); err != nil {
			return nil, err
		}
		pms = append(pms, pm)
	}
	return pms, rows.Err()
}

func (r *PaymentRepository) DeletePaymentMethod(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM payment_methods WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrPaymentMethodNotFound
	}
	return nil
}

func (r *PaymentRepository) GetUnpaidInvoiceAmount(ctx context.Context, userID uuid.UUID) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(i.amount), 0) FROM invoices i WHERE i.user_id = $1 AND i.status = 'pending'`, userID).Scan(&total)
	return total, err
}

func (r *PaymentRepository) DeductBalance(ctx context.Context, walletID uuid.UUID, amount float64) (float64, error) {
	var newBalance float64
	err := r.pool.QueryRow(ctx, `UPDATE wallets SET balance = balance - $1, updated_at = NOW() WHERE id = $2 AND balance >= $1 RETURNING balance`, amount, walletID).Scan(&newBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInsufficientBalance
		}
		return 0, err
	}
	return newBalance, nil
}

func (r *PaymentRepository) GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*model.Invoice, error) {
	var inv model.Invoice
	err := r.pool.QueryRow(ctx, `SELECT id, user_id, amount, status FROM invoices WHERE id = $1`, invoiceID).Scan(&inv.ID, &inv.UserID, &inv.Amount, &inv.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceNotFound
	}
	return &inv, err
}

func (r *PaymentRepository) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE invoices SET status = 'paid', paid_at = NOW(), updated_at = NOW() WHERE id = $1`, invoiceID)
	return err
}
