package leonardo

import (
	"encoding/json"
	"testing"
)

func TestListVerifiedModels(t *testing.T) {
	models := ListVerifiedModels()
	if len(models) != 23 {
		t.Fatalf("ListVerifiedModels() returned %d models, want 23", len(models))
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

func TestVerifiedImageStyleModels(t *testing.T) {
	tests := []struct{ slug, id string }{
		{"kino-xl", "aa77f04e-3eec-4034-9c07-d0f619684628"},
		{"concept-art", "dd29ac47-ea88-4720-8678-b8633245c09c"},
		{"graphic-design", "9d4ace10-25dd-42fd-a6be-a301a7ac614f"},
		{"illustrative-albedo", "2067ae52-33fd-4a82-bb92-c2c55e7d2786"},
	}
	for _, test := range tests {
		model, ok := ResolveByRequestModelSlug(test.slug)
		if !ok || model.ProviderModelID != test.id || model.Modality != ModelModalityImage || model.ImageCapabilities == nil || model.ImageCapabilities.MaxQuantity != 8 {
			t.Fatalf("ResolveByRequestModelSlug(%q) = %#v, %v", test.slug, model, ok)
		}
		if len(model.ImageCapabilities.AllowedQualities) != 2 || model.ImageCapabilities.AllowedQualities[0] != "low" || model.ImageCapabilities.AllowedQualities[1] != "high" {
			t.Fatalf("ResolveByRequestModelSlug(%q) qualities = %#v", test.slug, model.ImageCapabilities.AllowedQualities)
		}
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

func TestListVerifiedVideoModels(t *testing.T) {
	models := ListVerifiedVideoModels()
	want := map[string]string{
		"seedance-1.0-pro-fast":    "b959ecc2-a7f0-4618-9877-1bc45fc27570",
		"motion_2.0-fast":          "0a7a3eb2-3905-480b-a89a-2f3ffff545e7",
		"seedance-1.0-pro":         "728c9eac-b17d-47fe-b382-b9a28687fa85",
		"wan-2.7":                  "52884d8c-e2b9-4bb1-8ed5-927d390fe53a",
		"kling-video-o-3":          "0d5109cf-d256-4720-86d3-d8e5ff5a3ce2",
		"seedance-2.0":             "d30c33b2-c845-4734-8292-a638891332f9",
		"seedance-2.0-fast":        "696cdf87-86ba-4b31-b00b-afd7041697d8",
		"seedance-2.0-mini":        "43dc2a43-c0f1-4ab9-9eac-a1e56f2282e5",
		"kling-2.1":                "564204e1-996f-455b-a85b-fd35379da714",
		"kling-2.5":                "803f541e-35b1-4ac7-99d0-bd9b089feded",
		"kling-2.5-turbo-standard": "0340954a-1d54-4930-b648-7d4c052ba029",
		"kling-2.6":                "de8e0850-9511-492f-bd34-0a43e6b65a20",
		"kling-3.0":                "6c904469-5291-4043-b610-f53b50dfd6ff",
		"kling-3.0-turbo":          "b49a5ad1-e98b-4637-ab8c-d6e665b48c28",
		"kling-video-o-1":          "898c7f67-106c-42cd-9554-030572eda8b7",
	}
	if len(models) != len(want) {
		t.Fatalf("ListVerifiedVideoModels() returned %#v", models)
	}
	for _, model := range models {
		if want[model.RequestModelSlug] != model.ProviderModelID || model.Modality != ModelModalityVideo {
			t.Fatalf("ListVerifiedVideoModels() returned %#v", models)
		}
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
