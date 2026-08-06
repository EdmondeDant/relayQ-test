package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type leonardoImageInputRoundTripper func(*http.Request) (*http.Response, error)

func (f leonardoImageInputRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestLeonardoImageInputValidatesDetectedType(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 504)...)
	input, err := ReadLeonardoMultipartImage(bytes.NewReader(png), "fake.jpg", "image/png; charset=binary", int64(len(png)))
	require.NoError(t, err)
	require.Equal(t, "image/png", input.MIMEType)
	require.Equal(t, "png", input.Extension)
	require.Equal(t, "fake.jpg", input.FileName)

	_, err = ReadLeonardoMultipartImage(bytes.NewReader(png), "image.png", "image/jpeg", int64(len(png)))
	require.ErrorIs(t, err, ErrLeonardoImageInputInvalid)
}

func TestParseLeonardoImageDataURI(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 504)...)
	value := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	input, err := ParseLeonardoImageDataURI(value, int64(len(png)))
	require.NoError(t, err)
	require.Equal(t, png, input.Data)
	require.Equal(t, "png", input.Extension)

	_, err = ParseLeonardoImageDataURI("data:image/png;base64,%%%", 1024)
	require.ErrorIs(t, err, ErrLeonardoImageInputInvalid)
}

func TestLeonardoImageInputRejectsTooLargeAndNonImage(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 504)...)
	_, err := ReadLeonardoMultipartImage(bytes.NewReader(png), "image.png", "image/png", int64(len(png)-1))
	require.True(t, errors.Is(err, ErrLeonardoImageInputTooLarge))

	_, err = ReadLeonardoMultipartImage(bytes.NewReader([]byte("not an image")), "image.png", "image/png", 1024)
	require.ErrorIs(t, err, ErrLeonardoImageInputInvalid)
}

func TestDownloadLeonardoImageURLValidatesImage(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 504)...)
	client := &http.Client{Transport: leonardoImageInputRoundTripper(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "cdn.example", request.URL.Hostname())
		require.Empty(t, request.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"image/png"}}, Body: io.NopCloser(bytes.NewReader(png)), ContentLength: int64(len(png)), Request: request}, nil
	})}

	input, err := downloadLeonardoImageURL(context.Background(), "https://cdn.example/path/fake.jpg?signature=secret", int64(len(png)), client)
	require.NoError(t, err)
	require.Equal(t, "png", input.Extension)
	require.Equal(t, "fake.jpg", input.FileName)
}

func TestDownloadLeonardoImageURLRejectsUnsafeURLAndRedirect(t *testing.T) {
	called := false
	client := &http.Client{Transport: leonardoImageInputRoundTripper(func(request *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"http://127.0.0.1/private.png"}}, Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	})}
	for _, rawURL := range []string{"file:///etc/passwd", "http://localhost/image.png", "http://127.0.0.1/image.png", "http://user:password@example.com/image.png"} {
		_, err := downloadLeonardoImageURL(context.Background(), rawURL, 1024, client)
		require.ErrorIs(t, err, ErrLeonardoImageInputInvalid)
	}
	require.False(t, called)

	_, err := downloadLeonardoImageURL(context.Background(), "https://cdn.example/image.png", 1024, client)
	require.ErrorIs(t, err, ErrLeonardoImageInputInvalid)
}

func TestDownloadLeonardoImageURLRejectsOversizedBody(t *testing.T) {
	client := &http.Client{Transport: leonardoImageInputRoundTripper(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 1025))), ContentLength: -1, Request: request}, nil
	})}
	_, err := downloadLeonardoImageURL(context.Background(), "https://cdn.example/image.png", 1024, client)
	require.ErrorIs(t, err, ErrLeonardoImageInputTooLarge)
}
