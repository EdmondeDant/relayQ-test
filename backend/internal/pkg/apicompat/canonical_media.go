package apicompat

import "strings"

// CanonicalContentKind is the protocol-neutral media kind used by RelayQ's
// compatibility layer. Keeping this internal representation separate from
// provider-specific schemas prevents silent modality loss such as video being
// remapped to image_url.
type CanonicalContentKind string

const (
	CanonicalContentText  CanonicalContentKind = "text"
	CanonicalContentImage CanonicalContentKind = "image"
	CanonicalContentVideo CanonicalContentKind = "video"
	CanonicalContentAudio CanonicalContentKind = "audio"
	CanonicalContentFile  CanonicalContentKind = "file"
)

// CanonicalContentPart is a normalized content part independent of OpenAI
// Chat Completions, Responses, Gemini, Claude, or provider-specific payloads.
type CanonicalContentPart struct {
	Kind     CanonicalContentKind
	Text     string
	URL      string
	Data     string
	MIMEType string
	Detail   string
	Format   string
	UUID     string
}

// CanonicalContentPartsFromChat normalizes OpenAI-compatible chat content
// parts. It is intentionally conservative: it preserves media kinds instead of
// coercing unsupported kinds into images or text.
func CanonicalContentPartsFromChat(parts []ChatContentPart) []CanonicalContentPart {
	out := make([]CanonicalContentPart, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				out = append(out, CanonicalContentPart{Kind: CanonicalContentText, Text: p.Text})
			}
		case "image_url":
			if p.ImageURL != nil && p.ImageURL.URL != "" {
				out = append(out, CanonicalContentPart{
					Kind:   CanonicalContentImage,
					URL:    p.ImageURL.URL,
					Detail: p.ImageURL.Detail,
				})
			}
		case "video_url":
			if p.VideoURL != nil && p.VideoURL.URL != "" {
				out = append(out, CanonicalContentPart{Kind: CanonicalContentVideo, URL: p.VideoURL.URL})
			}
		case "audio_url":
			if p.AudioURL != nil && p.AudioURL.URL != "" {
				out = append(out, CanonicalContentPart{Kind: CanonicalContentAudio, URL: p.AudioURL.URL})
			}
		case "input_audio":
			if p.InputAudio != nil && (p.InputAudio.Data != "" || p.InputAudio.Format != "") {
				out = append(out, CanonicalContentPart{
					Kind:   CanonicalContentAudio,
					Data:   p.InputAudio.Data,
					Format: p.InputAudio.Format,
				})
			}
		}
	}
	return out
}

// CanonicalKindFromMIMEType maps provider-native media MIME types to RelayQ's
// canonical media kinds. This is intentionally explicit to avoid the new-api
// #4252 class of bug where video/* was accidentally converted to image_url.
func CanonicalKindFromMIMEType(mimeType string) CanonicalContentKind {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return CanonicalContentImage
	case strings.HasPrefix(mimeType, "video/"):
		return CanonicalContentVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return CanonicalContentAudio
	case mimeType != "":
		return CanonicalContentFile
	default:
		return ""
	}
}
