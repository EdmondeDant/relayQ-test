package leonardo

type VerifiedModel struct {
	DisplayName      string
	ProviderModelID  string
	RequestModelSlug string
}

var verifiedModels = []VerifiedModel{
	{
		DisplayName:      "FLUX Schnell",
		ProviderModelID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		RequestModelSlug: "flux-schnell",
	},
}

func ListVerifiedModels() []VerifiedModel {
	return append([]VerifiedModel(nil), verifiedModels...)
}

func ResolveByRequestModelSlug(slug string) (VerifiedModel, bool) {
	for _, model := range verifiedModels {
		if model.RequestModelSlug == slug {
			return model, true
		}
	}
	return VerifiedModel{}, false
}

func ResolveByProviderModelID(id string) (VerifiedModel, bool) {
	for _, model := range verifiedModels {
		if model.ProviderModelID == id {
			return model, true
		}
	}
	return VerifiedModel{}, false
}
