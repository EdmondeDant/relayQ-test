package service

import (
	"context"
	"encoding/json"
	"strings"
)

type LeonardoRawImageSource struct {
	Section string
	Index   int
	Value   string
}

func ParseLeonardoRawImageSources(body []byte) ([]LeonardoRawImageSource, error) {
	var request map[string]any
	if json.Unmarshal(body, &request) != nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	parameters, ok := request["parameters"].(map[string]any)
	if !ok {
		return nil, ErrLeonardoImageInputInvalid
	}
	guidances, hasGuidances := parameters["guidances"].(map[string]any)
	if !hasGuidances {
		return []LeonardoRawImageSource{}, nil
	}
	sources := []LeonardoRawImageSource{}
	for _, section := range []string{"content", "style", "image_reference", "start_frame", "end_frame"} {
		values, _ := guidances[section].([]any)
		for index, value := range values {
			reference, ok := value.(map[string]any)
			if !ok {
				continue
			}
			image, ok := reference["image"].(map[string]any)
			if !ok {
				continue
			}
			source, hasSource := image["source"].(string)
			_, hasID := image["id"]
			if !hasSource {
				continue
			}
			if hasID || strings.TrimSpace(source) == "" {
				return nil, ErrLeonardoImageInputInvalid
			}
			sources = append(sources, LeonardoRawImageSource{Section: section, Index: index, Value: strings.TrimSpace(source)})
		}
	}
	return sources, nil
}

func ParseLeonardoFluxImageSources(body []byte) ([]LeonardoRawImageSource, error) {
	return ParseLeonardoRawImageSources(body)
}

func ResolveLeonardoRawImageSource(ctx context.Context, source string, multipartImages map[string]*LeonardoImageInput, maxBytes int64) (*LeonardoImageInput, error) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "data:") {
		return ParseLeonardoImageDataURI(source, maxBytes)
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		return DownloadLeonardoImageURL(ctx, source, maxBytes)
	}
	if name, ok := strings.CutPrefix(source, "multipart://"); ok {
		image := multipartImages[strings.TrimSpace(name)]
		if image == nil {
			return nil, ErrLeonardoImageInputInvalid
		}
		return image, nil
	}
	return nil, ErrLeonardoImageInputInvalid
}

func SetLeonardoRawImageID(body []byte, source LeonardoRawImageSource, id string) ([]byte, error) {
	var request map[string]any
	if json.Unmarshal(body, &request) != nil || strings.TrimSpace(id) == "" {
		return nil, ErrLeonardoImageInputInvalid
	}
	parameters, ok := request["parameters"].(map[string]any)
	if !ok {
		return nil, ErrLeonardoImageInputInvalid
	}
	guidances, ok := parameters["guidances"].(map[string]any)
	if !ok {
		return nil, ErrLeonardoImageInputInvalid
	}
	values, ok := guidances[source.Section].([]any)
	if !ok || source.Index < 0 || source.Index >= len(values) {
		return nil, ErrLeonardoImageInputInvalid
	}
	reference, ok := values[source.Index].(map[string]any)
	if !ok {
		return nil, ErrLeonardoImageInputInvalid
	}
	image, ok := reference["image"].(map[string]any)
	if !ok || image["source"] != source.Value {
		return nil, ErrLeonardoImageInputInvalid
	}
	delete(image, "source")
	image["id"] = strings.TrimSpace(id)
	image["type"] = "UPLOADED"
	return json.Marshal(request)
}
