package repository

import (
	"context"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/usagelog"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type mediaUsageAuditRepository struct{ client *dbent.Client }

func NewMediaUsageAuditRepository(client *dbent.Client) service.MediaUsageAuditRepository {
	return &mediaUsageAuditRepository{client: client}
}

func (r *mediaUsageAuditRepository) CreateMediaUsageAudit(ctx context.Context, log *service.UsageLog) (bool, error) {
	if log == nil {
		return false, nil
	}
	requestedModel := strings.TrimSpace(log.RequestedModel)
	create := r.client.UsageLog.Create().
		SetUserID(log.UserID).
		SetAPIKeyID(log.APIKeyID).
		SetAccountID(log.AccountID).
		SetRequestID(strings.TrimSpace(log.RequestID)).
		SetModel(log.Model).
		SetNillableRequestedModel(&requestedModel).
		SetNillableUpstreamModel(log.UpstreamModel).
		SetNillableGroupID(log.GroupID).
		SetNillableMediaProductID(log.MediaProductID).
		SetNillableMediaOfferID(log.MediaOfferID).
		SetNillableUpstreamPlatform(log.UpstreamPlatform).
		SetNillableSourceGroupID(log.SourceGroupID).
		SetNillableTrustedCostAmount(log.TrustedCostAmount).
		SetNillableTrustedCostUnit(log.TrustedCostUnit).
		SetNillableTrustedCostSource(log.TrustedCostSource).
		SetNillableTrustedCostVersion(log.TrustedCostVersion).
		SetNillableCustomerPriceVersion(log.CustomerPriceVersion).
		SetActualCost(log.ActualCost).
		SetTotalCost(log.TotalCost).
		SetImageCount(log.ImageCount).
		SetNillableImageSize(log.ImageSize).
		SetNillableImageInputSize(log.ImageInputSize).
		SetNillableImageOutputSize(log.ImageOutputSize).
		SetNillableImageSizeSource(log.ImageSizeSource).
		SetNillableBillingMode(log.BillingMode)
	if log.ImageSizeBreakdown != nil {
		create.SetImageSizeBreakdown(log.ImageSizeBreakdown)
	}
	created, err := create.
		OnConflictColumns(usagelog.FieldRequestID, usagelog.FieldAPIKeyID).
		DoNothing().
		ID(ctx)
	if dbent.IsConstraintError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	log.ID = created
	return true, nil
}
