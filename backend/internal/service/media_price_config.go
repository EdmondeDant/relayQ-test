package service

func groupSearchPricePer1kFromAPIKey(apiKey *APIKey) *float64 {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	return apiKey.Group.SearchPricePer1k
}

func groupAudioPriceConfigFromAPIKey(apiKey *APIKey) *audioPriceConfig {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	return &audioPriceConfig{
		RealtimePerMin: apiKey.Group.AudioRealtimePricePerMin,
		TTSPerMChars:   apiKey.Group.AudioTTSPricePerMillionChars,
		STTPerHour:     apiKey.Group.AudioSTTPricePerHour,
	}
}
