package service

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

const (
	leonardoImageDownloadTimeout      = 30 * time.Second
	leonardoImageDownloadMaxRedirects = 5
)

var (
	ErrLeonardoImageInputInvalid  = errors.New("leonardo image input is invalid")
	ErrLeonardoImageInputTooLarge = errors.New("leonardo image input is too large")
)

type LeonardoImageInput struct {
	Data      []byte
	MIMEType  string
	Extension string
	FileName  string
}

func DownloadLeonardoImageURL(ctx context.Context, rawURL string, maxBytes int64) (*LeonardoImageInput, error) {
	return downloadLeonardoImageURL(ctx, rawURL, maxBytes, newSSRFSafeHTTPClient(leonardoImageDownloadTimeout))
}

func downloadLeonardoImageURL(ctx context.Context, rawURL string, maxBytes int64, baseClient *http.Client) (*LeonardoImageInput, error) {
	if ctx == nil || maxBytes <= 0 || baseClient == nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	parsed, err := validateLeonardoImageURL(rawURL)
	if err != nil {
		return nil, err
	}
	client := *baseClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= leonardoImageDownloadMaxRedirects {
			return ErrLeonardoImageInputInvalid
		}
		_, err := validateLeonardoImageURL(req.URL.String())
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	req.Header.Set("Accept", "image/jpeg, image/png, image/webp")
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrLeonardoImageInputInvalid
	}
	if resp.ContentLength > maxBytes {
		return nil, ErrLeonardoImageInputTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	fileName := path.Base(parsed.Path)
	if fileName == "." || fileName == "/" {
		fileName = ""
	}
	return validateLeonardoImageBytes(data, resp.Header.Get("Content-Type"), fileName, maxBytes)
}

func validateLeonardoImageURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || apicompat.IsPotentiallyUnsafeRemoteMediaURL(rawURL) {
		return nil, ErrLeonardoImageInputInvalid
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, ErrLeonardoImageInputInvalid
	}
	parsed.Fragment = ""
	return parsed, nil
}

func ParseLeonardoImageDataURI(value string, maxBytes int64) (*LeonardoImageInput, error) {
	metadata, payload, ok := strings.Cut(strings.TrimSpace(value), ",")
	if !ok || !strings.HasPrefix(strings.ToLower(metadata), "data:image/") || !strings.HasSuffix(strings.ToLower(metadata), ";base64") {
		return nil, ErrLeonardoImageInputInvalid
	}
	declared := metadata[len("data:") : len(metadata)-len(";base64")]
	if maxBytes <= 0 || int64(len(payload)) > (maxBytes+2)/3*4+4 {
		return nil, ErrLeonardoImageInputTooLarge
	}
	data, err := base64.StdEncoding.Strict().DecodeString(payload)
	if err != nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	return validateLeonardoImageBytes(data, declared, "", maxBytes)
}

func ReadLeonardoMultipartImage(reader io.Reader, fileName, declaredMIME string, maxBytes int64) (*LeonardoImageInput, error) {
	if reader == nil || maxBytes <= 0 {
		return nil, ErrLeonardoImageInputInvalid
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, ErrLeonardoImageInputInvalid
	}
	return validateLeonardoImageBytes(data, declaredMIME, fileName, maxBytes)
}

func validateLeonardoImageBytes(data []byte, declaredMIME, fileName string, maxBytes int64) (*LeonardoImageInput, error) {
	if len(data) == 0 {
		return nil, ErrLeonardoImageInputInvalid
	}
	if maxBytes <= 0 || int64(len(data)) > maxBytes {
		return nil, ErrLeonardoImageInputTooLarge
	}
	detected := http.DetectContentType(data)
	extension := ""
	switch detected {
	case "image/jpeg":
		extension = "jpg"
	case "image/png":
		extension = "png"
	case "image/webp":
		extension = "webp"
	default:
		return nil, ErrLeonardoImageInputInvalid
	}
	if strings.TrimSpace(declaredMIME) != "" {
		parsed, _, err := mime.ParseMediaType(declaredMIME)
		if err != nil {
			return nil, ErrLeonardoImageInputInvalid
		}
		if parsed == "image/jpg" {
			parsed = "image/jpeg"
		}
		if parsed != detected {
			return nil, ErrLeonardoImageInputInvalid
		}
	}
	return &LeonardoImageInput{Data: data, MIMEType: detected, Extension: extension, FileName: strings.TrimSpace(fileName)}, nil
}
