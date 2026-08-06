package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type leonardoInitImageClientFake struct {
	createCalls int
	uploadCalls int
}

func (f *leonardoInitImageClientFake) CreateInitImageUpload(context.Context, string) (*leonardo.InitImageUpload, error) {
	f.createCalls++
	return &leonardo.InitImageUpload{ID: "uploaded-1", URL: "https://upload.example", Fields: map[string]string{"key": "value"}}, nil
}

func (f *leonardoInitImageClientFake) UploadInitImage(context.Context, *leonardo.InitImageUpload, string, []byte) error {
	f.uploadCalls++
	return nil
}

func TestLeonardoImageUploadServiceUsesAccountScopedHashCache(t *testing.T) {
	server := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	service := NewLeonardoImageUploadService(NewLeonardoImageUploadCache(redisClient))
	client := &leonardoInitImageClientFake{}
	input := &LeonardoImageInput{Data: []byte("image"), Extension: "png", FileName: "image.png"}

	first, err := service.Upload(context.Background(), 1, client, input)
	require.NoError(t, err)
	second, err := service.Upload(context.Background(), 1, client, input)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 1, client.createCalls)
	require.Equal(t, 1, client.uploadCalls)

	_, err = service.Upload(context.Background(), 2, client, input)
	require.NoError(t, err)
	require.Equal(t, 2, client.createCalls)
}
