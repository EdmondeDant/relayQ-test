package leonardo

import "testing"

func TestListVerifiedModels(t *testing.T) {
	models := ListVerifiedModels()
	if len(models) != 1 {
		t.Fatalf("ListVerifiedModels() returned %d models, want 1", len(models))
	}
	want := VerifiedModel{
		DisplayName:      "FLUX Schnell",
		ProviderModelID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		RequestModelSlug: "flux-schnell",
	}
	if models[0] != want {
		t.Fatalf("ListVerifiedModels()[0] = %#v, want %#v", models[0], want)
	}

	models[0] = VerifiedModel{}
	if got := ListVerifiedModels()[0]; got != want {
		t.Fatalf("mutating returned models changed catalog to %#v, want %#v", got, want)
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
