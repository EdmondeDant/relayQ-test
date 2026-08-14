package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type MediaRuntimeSelection struct {
	Product        MediaCatalogProduct
	Price          MediaCatalogPrice
	Charge         decimal.Decimal
	RankedEligible []MediaOfferCandidate
	Skipped        []MediaOfferCandidate
}

func SelectMediaRuntime(product MediaCatalogProduct, request MediaCanonicalRequest, now time.Time) (MediaRuntimeSelection, error) {
	specKey := MediaSpecKey(request)
	price, found := mediaRuntimePrice(product.Prices, request.Operation, specKey)
	if !found {
		return MediaRuntimeSelection{}, ErrMediaCustomerPrice
	}
	quantity := mediaRequestQuantity(request.Fields)
	charge := price.UnitPriceUSD.Mul(decimal.NewFromInt(int64(quantity)))
	domainProduct := MediaProduct{ID: product.ID, PublicModel: product.PublicModel, Modality: product.Modality, Enabled: product.Enabled, CustomerPrice: MediaCustomerPrice{Basis: mediaPriceBasis(product.Modality), UnitPrice: price.UnitPriceUSD.InexactFloat64(), Currency: price.Currency, Version: price.Version}}
	offers := make([]MediaOffer, 0, len(product.Offers))
	allowed := make(map[int64]struct{}, len(product.Offers))
	for _, offer := range product.Offers {
		offers = append(offers, mediaRuntimeOffer(product.ID, offer))
		allowed[offer.SourceGroupID] = struct{}{}
	}
	fields := make(map[string]any, len(request.Fields))
	for key, value := range request.Fields {
		if key != "model" {
			fields[key] = value
		}
	}
	result := SelectMediaOffer(domainProduct, offers, MediaSelectRequest{ProductPublicModel: product.PublicModel, Modality: MediaModality(product.Modality), Op: request.Operation, Fields: fields, ProviderExtensionFields: mediaProviderExtensionFields(fields), N: quantity, SizeTier: mediaStringField(fields, "size", "resolution"), DurationSec: mediaFloatField(fields, "duration", "duration_seconds", "seconds"), AllowedSourceGroupIDs: allowed, Now: now})
	if result.Err != nil {
		return MediaRuntimeSelection{}, result.Err
	}
	return MediaRuntimeSelection{Product: product, Price: price, Charge: charge, RankedEligible: result.RankedEligible, Skipped: result.Skipped}, nil
}

func MediaSpecKey(request MediaCanonicalRequest) string {
	parts := []string{strings.ToLower(strings.TrimSpace(request.Operation))}
	keys := make([]string, 0, len(request.Fields))
	for key := range request.Fields {
		switch key {
		case "model", "prompt", "input", "image", "mask", "reference_image", "first_frame", "last_frame":
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		encoded, _ := json.Marshal(request.Fields[key])
		parts = append(parts, key+"="+string(encoded))
	}
	if _, ok := request.Fields["n"]; !ok {
		parts = append(parts, "n=1")
	}
	return strings.Join(parts, "|")
}

func mediaRuntimePrice(prices []MediaCatalogPrice, operation, specKey string) (MediaCatalogPrice, bool) {
	var selected MediaCatalogPrice
	found := false
	for _, price := range prices {
		if !price.Enabled || !strings.EqualFold(price.Operation, operation) || price.SpecKey != specKey {
			continue
		}
		if found {
			return MediaCatalogPrice{}, false
		}
		selected, found = price, true
	}
	return selected, found
}

func mediaRuntimeOffer(productID int64, offer MediaCatalogOffer) MediaOffer {
	return MediaOffer{ID: offer.ID, ProductID: productID, Provider: offer.Provider, SourceGroupID: offer.SourceGroupID, UpstreamModel: offer.UpstreamModel, Enabled: offer.Enabled, Priority: offer.Priority, Operations: offer.Operations, Capability: mediaRuntimeCapability(offer), Cost: mediaRuntimeCost(offer)}
}

func mediaRuntimeCapability(offer MediaCatalogOffer) MediaCapabilityProfile {
	result := MediaCapabilityProfile{Ops: offer.Operations, SupportedFields: map[string]MediaFieldCapability{}}
	value := offer.Capabilities
	if nested, ok := value["supported_fields"].(map[string]any); ok {
		for name, raw := range nested {
			capability := MediaFieldCapability{}
			if config, ok := raw.(map[string]any); ok {
				capability.Required, _ = config["required"].(bool)
				capability.Enum = mediaStringSlice(config["enum"])
				if max, ok := mediaNumber(config["max"]); ok {
					capability.Max = &max
				}
			}
			result.SupportedFields[name] = capability
		}
	}
	if ops := mediaStringSlice(value["ops"]); len(ops) > 0 {
		result.Ops = ops
	}
	result.Async, _ = value["async"].(bool)
	if max, ok := mediaNumber(value["max_n"]); ok {
		result.MaxN = int(max)
	}
	return result
}

func mediaRuntimeCost(offer MediaCatalogOffer) TrustedCostPolicy {
	rules := offer.CostRules
	policy := TrustedCostPolicy{Basis: mediaStringValueAny(rules["basis"]), UnitCost: mediaFloatValue(rules["unit_cost"]), Currency: strings.ToUpper(mediaStringValueAny(rules["currency"])), TrustState: "verified", VerifiedAt: offer.VerifiedAt, MaxAge: offer.ExpiresAt.Sub(offer.VerifiedAt), Source: offer.CostSource, Version: offer.CostVersion, Tiers: map[string]float64{}}
	if policy.Currency == "" {
		policy.Currency = "USD"
	}
	if tiers, ok := rules["tiers"].(map[string]any); ok {
		for key, value := range tiers {
			policy.Tiers[key] = mediaFloatValue(value)
		}
	}
	return policy
}

func mediaRequestQuantity(fields map[string]any) int {
	if value, ok := mediaNumber(fields["n"]); ok && value > 0 && value == float64(int(value)) {
		return int(value)
	}
	return 1
}

func mediaProviderExtensionFields(fields map[string]any) map[string]string {
	result := map[string]string{}
	for key := range fields {
		for _, provider := range []string{PlatformLeonardo, PlatformOpenAI} {
			if strings.HasPrefix(key, provider+"_") {
				result[key] = provider
			}
		}
	}
	return result
}

func mediaStringField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(mediaStringValueAny(fields[key])); value != "" {
			return value
		}
	}
	return ""
}

func mediaFloatField(fields map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := mediaNumber(fields[key]); ok {
			return value
		}
	}
	return 0
}

func mediaStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		if values, ok := value.([]string); ok {
			return values
		}
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		result = append(result, fmt.Sprint(item))
	}
	return result
}

func mediaStringValueAny(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func mediaFloatValue(value any) float64 {
	if number, ok := mediaNumber(value); ok {
		return number
	}
	number, _ := strconv.ParseFloat(fmt.Sprint(value), 64)
	return number
}

func mediaPriceBasis(modality string) string {
	if modality == "video" {
		return "per_video"
	}
	return "per_image"
}
