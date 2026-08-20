package service

import (
	"encoding/json"
	"errors"
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
)

func RequireLeonardoVerifiedVideoModel(model string) error {
	model = strings.TrimSpace(model)
	for _, verified := range leonardo.ListVerifiedVideoModels() {
		if verified.RequestModelSlug == model {
			return nil
		}
	}
	return ErrLeonardoNoVerifiedVideoModels
}

var (
	ErrLeonardoNoVerifiedVideoModels     = errors.New("no verified Leonardo video models are available")
	ErrLeonardoVideoParameterUnsupported = errors.New("leonardo video parameter is not supported by this model")
)

type LeonardoVideoParameters struct {
	Duration       *int
	Width          *int
	Height         *int
	Quantity       *int
	MotionHasAudio *bool
}

type LeonardoVideoParameterBinding struct {
	Duration       string
	Width          string
	Height         string
	Quantity       string
	MotionHasAudio string
}

type leonardoVideoSchemaProperty struct {
	Type    string            `json:"type"`
	Enum    []json.RawMessage `json:"enum"`
	Minimum *json.Number      `json:"minimum"`
	Maximum *json.Number      `json:"maximum"`
}

func BuildLeonardoVideoParameters(schema map[string]json.RawMessage, binding LeonardoVideoParameterBinding, input LeonardoVideoParameters) (map[string]any, error) {
	result := make(map[string]any)
	values := []struct {
		path  string
		value any
		set   bool
	}{
		{binding.Duration, valueOrNil(input.Duration), input.Duration != nil},
		{binding.Width, valueOrNil(input.Width), input.Width != nil},
		{binding.Height, valueOrNil(input.Height), input.Height != nil},
		{binding.Quantity, valueOrNil(input.Quantity), input.Quantity != nil},
		{binding.MotionHasAudio, valueOrNil(input.MotionHasAudio), input.MotionHasAudio != nil},
	}
	if (input.Width == nil) != (input.Height == nil) {
		return nil, ErrLeonardoVideoParameterUnsupported
	}
	for _, item := range values {
		if !item.set {
			continue
		}
		name := strings.TrimSpace(item.path)
		raw, ok := schema[name]
		if name == "" || !ok || !validLeonardoVideoSchemaValue(raw, item.value) {
			return nil, ErrLeonardoVideoParameterUnsupported
		}
		result[name] = item.value
	}
	return result, nil
}

func valueOrNil[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}

func validLeonardoVideoSchemaValue(raw json.RawMessage, value any) bool {
	var property leonardoVideoSchemaProperty
	if json.Unmarshal(raw, &property) != nil {
		return false
	}
	switch value.(type) {
	case int:
		if property.Type != "integer" && property.Type != "number" {
			return false
		}
	case bool:
		if property.Type != "boolean" {
			return false
		}
	default:
		return false
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return false
	}
	if len(property.Enum) > 0 {
		matched := false
		for _, enum := range property.Enum {
			if string(enum) == string(encoded) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	number, numeric := value.(int)
	if !numeric {
		return true
	}
	if property.Minimum != nil {
		minimum, err := property.Minimum.Float64()
		if err != nil || float64(number) < minimum {
			return false
		}
	}
	if property.Maximum != nil {
		maximum, err := property.Maximum.Float64()
		if err != nil || float64(number) > maximum || math.IsNaN(maximum) {
			return false
		}
	}
	return true
}
