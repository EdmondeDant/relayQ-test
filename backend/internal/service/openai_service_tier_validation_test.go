package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOpenAIServiceTierField(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
		err  bool
	}{
		{name: "omitted", body: `{"model":"gpt-5.6-sol"}`},
		{name: "null", body: `{"model":"gpt-5.6-sol","service_tier":null}`},
		{name: "fast canonicalizes", body: `{"service_tier":" Fast "}`, want: "priority"},
		{name: "official tiers", body: `{"service_tier":"scale"}`, want: "scale"},
		{name: "empty rejected", body: `{"service_tier":" "}`, err: true},
		{name: "unknown rejected", body: `{"service_tier":"turbo"}`, err: true},
		{name: "non string rejected", body: `{"service_tier":1}`, err: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateOpenAIServiceTierField([]byte(tt.body))
			if tt.err {
				require.Error(t, err)
				var tierErr *ErrInvalidOpenAIServiceTier
				require.True(t, errors.As(err, &tierErr))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
