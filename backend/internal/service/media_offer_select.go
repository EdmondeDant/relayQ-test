package service

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MediaModality string
type MediaProvider string
type MediaSkipReason string

const (
	MediaModalityImage MediaModality = "image"
	MediaModalityVideo MediaModality = "video"

	MediaSkipDisabled              MediaSkipReason = "disabled"
	MediaSkipProductMismatch       MediaSkipReason = "product_mismatch"
	MediaSkipSourceGroupNotAllowed MediaSkipReason = "source_group_not_allowed"
	MediaSkipUnsupportedOp         MediaSkipReason = "unsupported_op"
	MediaSkipUnsupportedField      MediaSkipReason = "unsupported_field"
	MediaSkipFieldEnumMismatch     MediaSkipReason = "field_enum_mismatch"
	MediaSkipUntrustedCost         MediaSkipReason = "untrusted_cost"
	MediaSkipCostIncomplete        MediaSkipReason = "cost_incomplete"
)

var (
	ErrMediaProductNotFound = errors.New("media product not found")
	ErrMediaCustomerPrice   = errors.New("media customer price unavailable")
	ErrNoTrustedMediaOffer  = errors.New("no trusted media offer")
)

type MediaProduct struct {
	ID            int64
	PublicModel   string
	Modality      string
	CustomerPrice MediaCustomerPrice
	Enabled       bool
}

type MediaOffer struct {
	ID               int64
	ProductID        int64
	Provider         string
	SourceGroupID    int64
	SourceAccountID  *int64
	UpstreamModel    string
	Enabled          bool
	PriorityTiebreak int
	Priority         int
	Operations       []string
	Capability       MediaCapabilityProfile
	Cost             TrustedCostPolicy
}

type MediaCustomerPrice struct {
	Basis     string
	UnitPrice float64
	Currency  string
	Version   string
	Tiers     map[string]float64
}

type MediaCapabilityProfile struct {
	Ops             []string
	SupportedFields map[string]MediaFieldCapability
	Async           bool
	MaxN            int
}

type MediaFieldCapability struct {
	Required bool
	Enum     []string
	Max      *float64
}

type TrustedCostPolicy struct {
	Basis      string
	UnitCost   float64
	Tiers      map[string]float64
	Currency   string
	TrustState string
	VerifiedAt time.Time
	MaxAge     time.Duration
	Source     string
	Version    string
}

type MediaSelectRequest struct {
	ProductPublicModel      string
	Modality                MediaModality
	Op                      string
	Fields                  map[string]any
	ProviderExtensionFields map[string]string
	N                       int
	SizeTier                string
	DurationSec             float64
	AllowedSourceGroupIDs   map[int64]struct{}
	Now                     time.Time
}

type MediaOfferCandidate struct {
	Offer          MediaOffer
	TrustedCost    float64
	CustomerCharge float64
	Quantity       int
	SkipReason     MediaSkipReason
	SkipDetail     string
}

type MediaSelectResult struct {
	Product        MediaProduct
	Selected       *MediaOfferCandidate
	RankedEligible []MediaOfferCandidate
	Skipped        []MediaOfferCandidate
	Err            error
}

func SelectMediaOffer(product MediaProduct, offers []MediaOffer, req MediaSelectRequest) MediaSelectResult {
	result := MediaSelectResult{Product: product}
	if !product.Enabled || product.PublicModel != req.ProductPublicModel || product.Modality != string(req.Modality) {
		result.Err = ErrMediaProductNotFound
		return result
	}
	charge, err := mediaCustomerCharge(product.CustomerPrice, req)
	if err != nil {
		result.Err = err
		return result
	}
	for _, offer := range offers {
		candidate := MediaOfferCandidate{Offer: offer, CustomerCharge: charge, Quantity: req.N}
		if reason, detail := mediaOfferEligibility(product, offer, req); reason != "" {
			candidate.SkipReason, candidate.SkipDetail = reason, detail
			result.Skipped = append(result.Skipped, candidate)
			continue
		}
		cost, reason, detail := mediaTrustedCost(offer.Cost, req)
		if reason != "" {
			candidate.SkipReason, candidate.SkipDetail = reason, detail
			result.Skipped = append(result.Skipped, candidate)
			continue
		}
		candidate.TrustedCost = cost
		result.RankedEligible = append(result.RankedEligible, candidate)
	}
	sort.SliceStable(result.RankedEligible, func(i, j int) bool {
		left, right := result.RankedEligible[i], result.RankedEligible[j]
		if left.TrustedCost != right.TrustedCost {
			return left.TrustedCost < right.TrustedCost
		}
		if left.Offer.Priority != right.Offer.Priority {
			return left.Offer.Priority < right.Offer.Priority
		}
		return left.Offer.ID < right.Offer.ID
	})
	if len(result.RankedEligible) == 0 {
		result.Err = ErrNoTrustedMediaOffer
		return result
	}
	selected := result.RankedEligible[0]
	result.Selected = &selected
	return result
}

func mediaOfferEligibility(product MediaProduct, offer MediaOffer, req MediaSelectRequest) (MediaSkipReason, string) {
	if !offer.Enabled {
		return MediaSkipDisabled, "offer disabled"
	}
	if offer.ProductID != product.ID {
		return MediaSkipProductMismatch, "offer product does not match"
	}
	if _, ok := req.AllowedSourceGroupIDs[offer.SourceGroupID]; !ok {
		return MediaSkipSourceGroupNotAllowed, "source group is not allowed"
	}
	operations := offer.Capability.Ops
	if len(operations) == 0 {
		operations = offer.Operations
	}
	if !slices.Contains(operations, req.Op) {
		return MediaSkipUnsupportedOp, req.Op
	}
	if req.N <= 0 || offer.Capability.MaxN > 0 && req.N > offer.Capability.MaxN {
		return MediaSkipUnsupportedField, "n"
	}
	for name, capability := range offer.Capability.SupportedFields {
		if capability.Required {
			if _, ok := req.Fields[name]; !ok {
				return MediaSkipUnsupportedField, name
			}
		}
	}
	for name, value := range req.Fields {
		capability, ok := offer.Capability.SupportedFields[name]
		if !ok {
			if provider, extension := req.ProviderExtensionFields[name]; extension && provider != offer.Provider {
				continue
			}
			return MediaSkipUnsupportedField, name
		}
		if len(capability.Enum) > 0 && !slices.Contains(capability.Enum, fmt.Sprint(value)) {
			return MediaSkipFieldEnumMismatch, name
		}
		if capability.Max != nil {
			number, ok := mediaNumber(value)
			if !ok || number > *capability.Max {
				return MediaSkipUnsupportedField, name
			}
		}
	}
	return "", ""
}

func mediaTrustedCost(policy TrustedCostPolicy, req MediaSelectRequest) (float64, MediaSkipReason, string) {
	if policy.TrustState != "verified" || policy.VerifiedAt.IsZero() || policy.MaxAge <= 0 || req.Now.After(policy.VerifiedAt.Add(policy.MaxAge)) {
		return 0, MediaSkipUntrustedCost, policy.TrustState
	}
	quantity := float64(req.N)
	var cost float64
	switch policy.Basis {
	case "per_image", "per_video":
		cost = policy.UnitCost * quantity
	case "per_request":
		cost = policy.UnitCost
	case "per_video_second":
		cost = policy.UnitCost * req.DurationSec * quantity
	case "tiered":
		unit, ok := policy.Tiers[mediaTierKey(req)]
		if !ok {
			return 0, MediaSkipCostIncomplete, mediaTierKey(req)
		}
		cost = unit * quantity
	default:
		return 0, MediaSkipCostIncomplete, policy.Basis
	}
	if cost <= 0 || math.IsInf(cost, 0) || math.IsNaN(cost) {
		return 0, MediaSkipCostIncomplete, policy.Basis
	}
	return cost, "", ""
}

func mediaCustomerCharge(price MediaCustomerPrice, req MediaSelectRequest) (float64, error) {
	quantity := float64(req.N)
	var charge float64
	switch price.Basis {
	case "per_image", "per_video":
		charge = price.UnitPrice * quantity
	case "per_request":
		charge = price.UnitPrice
	case "tiered":
		unit, ok := price.Tiers[mediaTierKey(req)]
		if !ok {
			return 0, ErrMediaCustomerPrice
		}
		charge = unit * quantity
	default:
		return 0, ErrMediaCustomerPrice
	}
	if charge <= 0 || strings.TrimSpace(price.Currency) == "" || math.IsInf(charge, 0) || math.IsNaN(charge) {
		return 0, ErrMediaCustomerPrice
	}
	return charge, nil
}

func mediaTierKey(req MediaSelectRequest) string {
	if req.SizeTier != "" {
		return req.SizeTier
	}
	return strconv.FormatFloat(req.DurationSec, 'f', -1, 64)
}

func mediaNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case float64:
		return number, true
	case float32:
		return float64(number), true
	default:
		return 0, false
	}
}
