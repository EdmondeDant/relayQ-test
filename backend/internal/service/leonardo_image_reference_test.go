package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/stretchr/testify/require"
)

func TestBuildLeonardoImageReferenceGuidance(t *testing.T) {
	capability := &LeonardoImageReferenceCapability{MaxItems: 2, AllowedStrengths: []string{"LOW", "MID", "HIGH"}, StrengthRequired: true}
	references := []leonardo.ImageReference{
		{Image: leonardo.ImageReferenceImage{ID: " uploaded-1 ", Type: "UPLOADED"}, Strength: "MID"},
		{Image: leonardo.ImageReferenceImage{ID: "generated-1", Type: "GENERATED"}, Strength: "HIGH"},
	}

	guidance, err := BuildLeonardoImageReferenceGuidance(references, capability)
	require.NoError(t, err)
	result, ok := guidance["image_reference"].([]leonardo.ImageReference)
	require.True(t, ok)
	require.Equal(t, "uploaded-1", result[0].Image.ID)
	require.Equal(t, references[1], result[1])
	require.Equal(t, " uploaded-1 ", references[0].Image.ID)
}

func TestBuildLeonardoImageReferenceGuidanceRejectsUnsupportedValues(t *testing.T) {
	capability := &LeonardoImageReferenceCapability{MaxItems: 1, AllowedStrengths: []string{"MID"}, StrengthRequired: true}
	for _, references := range [][]leonardo.ImageReference{
		{{Image: leonardo.ImageReferenceImage{ID: "", Type: "UPLOADED"}, Strength: "MID"}},
		{{Image: leonardo.ImageReferenceImage{ID: "id", Type: "INIT"}, Strength: "MID"}},
		{{Image: leonardo.ImageReferenceImage{ID: "id", Type: "UPLOADED"}, Strength: "LOW"}},
		{{Image: leonardo.ImageReferenceImage{ID: "1", Type: "UPLOADED"}, Strength: "MID"}, {Image: leonardo.ImageReferenceImage{ID: "2", Type: "UPLOADED"}, Strength: "MID"}},
	} {
		_, err := BuildLeonardoImageReferenceGuidance(references, capability)
		require.ErrorIs(t, err, ErrLeonardoImageReferenceInvalid)
	}
	_, err := BuildLeonardoImageReferenceGuidance([]leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "id", Type: "UPLOADED"}, Strength: "MID"}}, nil)
	require.ErrorIs(t, err, ErrLeonardoImageReferenceInvalid)
}

func TestLeonardoImageReferenceCapabilities(t *testing.T) {
	gpt := LeonardoImageReferenceCapabilityForModel("gpt-image-2")
	require.NotNil(t, gpt)
	require.False(t, gpt.StrengthRequired)
	guidance, err := BuildLeonardoImageReferenceGuidance([]leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "uploaded", Type: "UPLOADED"}}}, gpt)
	require.NoError(t, err)
	encoded, err := json.Marshal(guidance)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "strength")
	_, err = BuildLeonardoImageReferenceGuidance([]leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "uploaded", Type: "UPLOADED"}, Strength: "MID"}}, gpt)
	require.ErrorIs(t, err, ErrLeonardoImageReferenceInvalid)

	nano := LeonardoImageReferenceCapabilityForModel("nano-banana-2")
	require.NotNil(t, nano)
	require.True(t, nano.StrengthRequired)
	guidance, err = BuildLeonardoImageReferenceGuidance([]leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "uploaded", Type: "UPLOADED"}}}, nano)
	require.NoError(t, err)
	result, ok := guidance["image_reference"].([]leonardo.ImageReference)
	require.True(t, ok)
	require.Equal(t, "MID", result[0].Strength)
}

func TestBuildLeonardoFluxGuidances(t *testing.T) {
	guidance, err := BuildLeonardoFluxGuidances("flux-schnell", LeonardoFluxGuidances{
		Content: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: " content-1 ", Type: "UPLOADED"}}},
		Style:   []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "style-1", Type: "GENERATED"}, Strength: "ULTRA"}},
	})
	require.NoError(t, err)
	require.Equal(t, []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "content-1", Type: "UPLOADED"}, Strength: "MID"}}, guidance["content"])
	require.Equal(t, []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "style-1", Type: "GENERATED"}, Strength: "ULTRA"}}, guidance["style"])
	require.NotContains(t, guidance, "image_reference")
}

func TestBuildLeonardoFluxGuidancesRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		model     string
		guidances LeonardoFluxGuidances
	}{
		{model: "unknown", guidances: LeonardoFluxGuidances{Content: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "id", Type: "UPLOADED"}, Strength: "MID"}}}},
		{model: "flux-schnell", guidances: LeonardoFluxGuidances{Content: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "id", Type: "UPLOADED"}, Strength: "ULTRA"}}}},
		{model: "flux-schnell", guidances: LeonardoFluxGuidances{Style: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "id", Type: "OTHER"}, Strength: "MID"}}}},
		{model: "flux-schnell", guidances: LeonardoFluxGuidances{Content: []leonardo.ImageReference{{Image: leonardo.ImageReferenceImage{ID: "1", Type: "UPLOADED"}, Strength: "MID"}, {Image: leonardo.ImageReferenceImage{ID: "2", Type: "UPLOADED"}, Strength: "MID"}}}},
	}
	for _, test := range tests {
		_, err := BuildLeonardoFluxGuidances(test.model, test.guidances)
		require.ErrorIs(t, err, ErrLeonardoImageReferenceInvalid)
	}
}
