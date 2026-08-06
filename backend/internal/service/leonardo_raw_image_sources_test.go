package service

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeonardoRawImageSources(t *testing.T) {
	body := []byte(`{"model":"flux-schnell","parameters":{"prompt":"cat","guidances":{"content":[{"image":{"source":"multipart://content_1","keep":"yes"},"strength":"HIGH"}],"style":[{"image":{"id":"existing","type":"GENERATED"}}]},"unknown":1}}`)
	sources, err := ParseLeonardoFluxImageSources(body)
	require.NoError(t, err)
	require.Equal(t, []LeonardoRawImageSource{{Section: "content", Index: 0, Value: "multipart://content_1"}}, sources)

	updated, err := SetLeonardoRawImageID(body, sources[0], "uploaded-1")
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"flux-schnell","parameters":{"prompt":"cat","guidances":{"content":[{"image":{"id":"uploaded-1","type":"UPLOADED","keep":"yes"},"strength":"HIGH"}],"style":[{"image":{"id":"existing","type":"GENERATED"}}]},"unknown":1}}`, string(updated))
}

func TestLeonardoRawImageReferenceSource(t *testing.T) {
	body := []byte(`{"model":"nano-banana-2","parameters":{"prompt":"cat","guidances":{"image_reference":[{"image":{"source":"multipart://image"},"strength":"MID"}]}}}`)
	sources, err := ParseLeonardoRawImageSources(body)
	require.NoError(t, err)
	require.Equal(t, []LeonardoRawImageSource{{Section: "image_reference", Index: 0, Value: "multipart://image"}}, sources)
	updated, err := SetLeonardoRawImageID(body, sources[0], "uploaded")
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"nano-banana-2","parameters":{"prompt":"cat","guidances":{"image_reference":[{"image":{"id":"uploaded","type":"UPLOADED"},"strength":"MID"}]}}}`, string(updated))
}

func TestResolveLeonardoRawImageSource(t *testing.T) {
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	require.NoError(t, err)
	input := &LeonardoImageInput{Data: png, MIMEType: "image/png", Extension: "png"}
	resolved, err := ResolveLeonardoRawImageSource(context.Background(), "multipart://image", map[string]*LeonardoImageInput{"image": input}, 20<<20)
	require.NoError(t, err)
	require.Same(t, input, resolved)
}

func TestParseLeonardoFluxImageSourcesRejectsSourceWithID(t *testing.T) {
	_, err := ParseLeonardoFluxImageSources([]byte(`{"parameters":{"guidances":{"content":[{"image":{"id":"existing","source":"https://example.com/image.png"}}]}}}`))
	require.ErrorIs(t, err, ErrLeonardoImageInputInvalid)
}
