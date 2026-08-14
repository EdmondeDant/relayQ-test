package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type mediaFundsRepoStub struct {
	reserved MediaFundsReserveRequest
	settled  MediaFundsTransitionRequest
	released MediaFundsTransitionRequest
}

func (r *mediaFundsRepoStub) Reserve(_ context.Context, request MediaFundsReserveRequest) (*MediaFundsReservation, error) {
	r.reserved = request
	return &MediaFundsReservation{Reference: "hold-1", UserID: request.UserID, PublicID: request.PublicID, ProductID: request.ProductID, Amount: request.Amount, PriceVersion: request.PriceVersion, Status: "reserved"}, nil
}

func (r *mediaFundsRepoStub) Settle(_ context.Context, request MediaFundsTransitionRequest) error {
	r.settled = request
	return nil
}

func (r *mediaFundsRepoStub) Release(_ context.Context, request MediaFundsTransitionRequest) error {
	r.released = request
	return nil
}

func TestMediaFundsServiceValidatesAndDelegates(t *testing.T) {
	repo := &mediaFundsRepoStub{}
	svc := NewMediaFundsService(repo)
	amount := decimal.RequireFromString("0.1234567890")
	reservation, err := svc.Reserve(context.Background(), MediaFundsReserveRequest{UserID: 1, PublicID: " job-1 ", ProductID: 2, Amount: amount, PriceVersion: " v1 "})
	require.NoError(t, err)
	require.Equal(t, "job-1", repo.reserved.PublicID)
	require.Equal(t, "v1", repo.reserved.PriceVersion)
	require.Equal(t, "hold-1", reservation.Reference)
	transition := MediaFundsTransitionRequest{UserID: 1, PublicID: " job-1 ", Reference: " hold-1 ", Amount: amount}
	require.NoError(t, svc.Settle(context.Background(), transition))
	require.NoError(t, svc.Release(context.Background(), transition))
	require.Equal(t, "job-1", repo.settled.PublicID)
	require.Equal(t, "hold-1", repo.released.Reference)
	_, err = svc.Reserve(context.Background(), MediaFundsReserveRequest{})
	require.ErrorIs(t, err, ErrMediaReservationInvalid)
}

type mediaUsageAuditRepoStub struct{ log *UsageLog }

func (r *mediaUsageAuditRepoStub) CreateMediaUsageAudit(_ context.Context, log *UsageLog) (bool, error) {
	r.log = log
	return true, nil
}

func TestMediaUsageAuditServiceMapsDraft(t *testing.T) {
	accountID, customerGroupID := int64(8), int64(3)
	repo := &mediaUsageAuditRepoStub{}
	svc := NewMediaUsageAuditService(repo)
	inserted, err := svc.Write(context.Background(), UsageLogDraft{RequestID: "req-1", APIKeyID: 2, UserID: 1, CustomerGroupID: &customerGroupID, AccountID: &accountID, RequestedModel: "relay-video", UpstreamModel: "vendor-video", MediaType: "video", ImageCount: 1, ActualCost: 0.2, ProductID: 4, OfferID: 5, UpstreamPlatform: "leonardo", SourceGroupID: 6, TrustedCost: 0.05, TrustedCostUnit: "USD", TrustedCostSource: "vendor", TrustedCostVersion: "cost-v1", CustomerPriceVersion: "price-v1"})
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, int64(4), *repo.log.MediaProductID)
	require.Equal(t, int64(5), *repo.log.MediaOfferID)
	require.Equal(t, "leonardo", *repo.log.UpstreamPlatform)
	require.Equal(t, "price-v1", *repo.log.CustomerPriceVersion)
	require.Equal(t, string(BillingModePerRequest), *repo.log.BillingMode)
}
