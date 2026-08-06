package leonardo

import (
	"encoding/json"
	"testing"
)

func TestListVerifiedModels(t *testing.T) {
	models := ListVerifiedModels()
	if len(models) != 4 {
		t.Fatalf("ListVerifiedModels() returned %d models, want 4", len(models))
	}
	want := VerifiedModel{
		DisplayName:      "FLUX Schnell",
		ProviderModelID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		RequestModelSlug: "flux-schnell",
		Modality:         ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{
			MinWidth: 32, MaxWidth: 2048, MinHeight: 32, MaxHeight: 2048, Multiple: 8,
			Content: &VerifiedImageGuidance{MaxItems: 1, AllowedStrengths: []string{"LOW", "MID", "HIGH"}, AllowedImageTypes: []string{"INIT", "GENERATION", "UPLOADED", "GENERATED", "VARIATION"}, DefaultStrength: "MID"},
			Style:   &VerifiedImageGuidance{MaxItems: 1, AllowedStrengths: []string{"LOW", "MID", "HIGH", "ULTRA", "MAX"}, AllowedImageTypes: []string{"INIT", "GENERATION", "UPLOADED", "GENERATED", "VARIATION"}, DefaultStrength: "MID"},
		},
	}
	if models[0].DisplayName != want.DisplayName || models[0].ProviderModelID != want.ProviderModelID || models[0].RequestModelSlug != want.RequestModelSlug || models[0].Modality != want.Modality {
		t.Fatalf("ListVerifiedModels()[0] = %#v, want %#v", models[0], want)
	}
	if models[0].ImageCapabilities == nil || models[0].ImageCapabilities.Content == nil || models[0].ImageCapabilities.Style == nil || models[0].ImageCapabilities.MaxWidth != 2048 || models[0].ImageCapabilities.Content.MaxItems != 1 || models[0].ImageCapabilities.Style.MaxItems != 1 {
		t.Fatalf("ListVerifiedModels()[0] capabilities = %#v", models[0].ImageCapabilities)
	}

	models[0] = VerifiedModel{}
	if got := ListVerifiedModels()[0]; got.RequestModelSlug != want.RequestModelSlug || got.ImageCapabilities == nil {
		t.Fatalf("mutating returned models changed catalog to %#v, want %#v", got, want)
	}
	models = ListVerifiedModels()
	models[0].ImageCapabilities.Content.AllowedStrengths[0] = "BROKEN"
	if got := ListVerifiedModels()[0].ImageCapabilities.Content.AllowedStrengths[0]; got != "LOW" {
		t.Fatalf("mutating returned capabilities changed catalog to %q", got)
	}
}

func TestVerifiedImageModels(t *testing.T) {
	tests := []struct {
		slug, id string
		strength bool
	}{
		{"gpt-image-2", "135b2740-a20b-48c8-8f86-6f68199e06c5", false},
		{"nano-banana-2", "7418e71f-4133-4e1b-9895-bee19f48f2ce", true},
		{"nano-banana-2-lite", "21278dfe-ac26-4292-82e0-8e588373a30c", true},
	}
	for _, test := range tests {
		model, ok := ResolveByRequestModelSlug(test.slug)
		if !ok || model.ProviderModelID != test.id || model.ImageCapabilities == nil || model.ImageCapabilities.ImageReference == nil || model.ImageCapabilities.ImageReference.MaxItems != 6 || model.ImageCapabilities.ImageReference.StrengthRequired != test.strength {
			t.Fatalf("ResolveByRequestModelSlug(%q) = %#v, %v", test.slug, model, ok)
		}
		if resolved, ok := ResolveByProviderModelID(test.id); !ok || resolved.RequestModelSlug != test.slug {
			t.Fatalf("ResolveByProviderModelID(%q) = %#v, %v", test.id, resolved, ok)
		}
	}
	models := ListVerifiedModels()
	models[1].ImageCapabilities.ImageReference.AllowedImageTypes[0] = "BROKEN"
	if got := ListVerifiedModels()[1].ImageCapabilities.ImageReference.AllowedImageTypes[0]; got != "UPLOADED" {
		t.Fatalf("mutating image reference capabilities changed catalog to %q", got)
	}
}

func TestValidateSyncedModelRequiresStrictSchema(t *testing.T) {
	strict := false
	validSchema, err := json.Marshal(ParameterSchema{Type: "object", Properties: map[string]json.RawMessage{"prompt": json.RawMessage(`{"type":"string"}`)}, Required: []string{"prompt"}, AdditionalProperties: &strict})
	if err != nil {
		t.Fatal(err)
	}
	model, ok := ValidateSyncedModel(Model{ID: "1dd50843-d653-4516-a8e3-f0238ee453ff", Parameters: validSchema})
	if !ok || model.Modality != ModelModalityImage {
		t.Fatalf("ValidateSyncedModel() = %#v, %v", model, ok)
	}
	for _, parameters := range []json.RawMessage{nil, json.RawMessage(`{"type":"array","properties":{},"additionalProperties":false}`), json.RawMessage(`{"type":"object","properties":{},"additionalProperties":true}`), json.RawMessage(`{"type":"object","properties":{}}`)} {
		if _, ok := ValidateSyncedModel(Model{ID: "1dd50843-d653-4516-a8e3-f0238ee453ff", Parameters: parameters}); ok {
			t.Fatalf("ValidateSyncedModel() accepted %s", parameters)
		}
	}
	if _, ok := ValidateSyncedModel(Model{ID: "unknown", Parameters: validSchema}); ok {
		t.Fatal("ValidateSyncedModel() accepted unknown model")
	}
}

func TestListVerifiedVideoModelsIsEmptyUntilEvidenceExists(t *testing.T) {
	if models := ListVerifiedVideoModels(); len(models) != 0 {
		t.Fatalf("ListVerifiedVideoModels() returned %#v", models)
	}
}

func TestListVerifiedAudioModelsIsEmptyUntilEvidenceExists(t *testing.T) {
	if models := ListVerifiedAudioModels(); len(models) != 0 {
		t.Fatalf("ListVerifiedAudioModels() returned %#v", models)
	}
}

func TestListVerified3DModelsIsEmptyUntilEvidenceExists(t *testing.T) {
	if models := ListVerified3DModels(); len(models) != 0 {
		t.Fatalf("ListVerified3DModels() returned %#v", models)
	}
}

func TestResolveByRequestModelSlug(t *testing.T) {
	model, ok := ResolveByRequestModelSlug("flux-schnell")
	if !ok {
		t.Fatal("ResolveVerifiedModelBySlug() did not resolve verified slug")
	}
	if model.ProviderModelID != "1dd50843-d653-4516-a8e3-f0238ee453ff" {
		t.Fatalf("ResolveByRequestModelSlug() ProviderModelID = %q", model.ProviderModelID)
	}

	for _, value := range []string{
		"unknown-slug",
		"FLUX Schnell",
		"1dd50843-d653-4516-a8e3-f0238ee453ff",
		"Flux-Schnell",
		" flux-schnell ",
	} {
		if _, ok := ResolveByRequestModelSlug(value); ok {
			t.Fatalf("ResolveByRequestModelSlug(%q) unexpectedly resolved", value)
		}
	}
}

func TestResolveByProviderModelID(t *testing.T) {
	model, ok := ResolveByProviderModelID("1dd50843-d653-4516-a8e3-f0238ee453ff")
	if !ok {
		t.Fatal("ResolveVerifiedModelByUUID() did not resolve verified UUID")
	}
	if model.RequestModelSlug != "flux-schnell" {
		t.Fatalf("ResolveByProviderModelID() RequestModelSlug = %q", model.RequestModelSlug)
	}

	for _, value := range []string{
		"00000000-0000-0000-0000-000000000000",
		"FLUX Schnell",
		"flux-schnell",
		"1DD50843-D653-4516-A8E3-F0238EE453FF",
		" 1dd50843-d653-4516-a8e3-f0238ee453ff ",
	} {
		if _, ok := ResolveByProviderModelID(value); ok {
			t.Fatalf("ResolveByProviderModelID(%q) unexpectedly resolved", value)
		}
	}
}
