package service

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/shopspring/decimal"
)

var ErrLeonardoImageQuoteBalanceInvalid = errors.New("leonardo image quote balance is invalid")

type LeonardoImageQuoteUserReader interface {
	GetByID(ctx context.Context, userID int64) (*User, error)
}

type LeonardoImageQuoteUserBalanceReader struct {
	users LeonardoImageQuoteUserReader
}

var _ LeonardoImageQuoteBalanceReader = (*LeonardoImageQuoteUserBalanceReader)(nil)

func NewLeonardoImageQuoteUserBalanceReader(users LeonardoImageQuoteUserReader) *LeonardoImageQuoteUserBalanceReader {
	return &LeonardoImageQuoteUserBalanceReader{users: users}
}

func (r *LeonardoImageQuoteUserBalanceReader) GetAvailableBalanceUSD(ctx context.Context, userID int64) (decimal.Decimal, error) {
	if err := ctx.Err(); err != nil {
		return decimal.Zero, err
	}
	if r == nil || r.users == nil || userID <= 0 {
		return decimal.Zero, ErrLeonardoImageQuoteBalanceInvalid
	}
	user, err := r.users.GetByID(ctx, userID)
	if err != nil {
		return decimal.Zero, err
	}
	if user == nil || user.ID != userID || math.IsNaN(user.Balance) || math.IsInf(user.Balance, 0) {
		return decimal.Zero, ErrLeonardoImageQuoteBalanceInvalid
	}
	formatted := strconv.FormatFloat(user.Balance, 'f', -1, 64)
	balance, err := decimal.NewFromString(formatted)
	if err != nil {
		return decimal.Zero, ErrLeonardoImageQuoteBalanceInvalid
	}
	return balance, nil
}
