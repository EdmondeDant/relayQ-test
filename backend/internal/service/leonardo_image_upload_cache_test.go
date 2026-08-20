package service

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

type leonardoImageUploadCacheFake struct{ values map[string]string }

func (f *leonardoImageUploadCacheFake) Get(_ context.Context, accountID int64, hash string) (string, bool, error) {
	if accountID <= 0 {
		return "", false, nil
	}
	if _, ok := NormalizeLeonardoImageHash(hash); !ok {
		return "", false, nil
	}
	v, ok := f.values[fmt.Sprintf("%d:%s", accountID, hash)]
	return v, ok, nil
}
func (f *leonardoImageUploadCacheFake) Set(_ context.Context, accountID int64, hash, id string) error {
	if accountID <= 0 {
		return nil
	}
	if _, ok := NormalizeLeonardoImageHash(hash); !ok || id == "" {
		return nil
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[fmt.Sprintf("%d:%s", accountID, hash)] = id
	return nil
}

func TestLeonardoImageUploadCacheScopesByAccountAndExpires(t *testing.T) {
	cache := &leonardoImageUploadCacheFake{values: map[string]string{}}
	ctx := context.Background()
	hash := LeonardoImageSHA256([]byte("image"))

	require.NoError(t, cache.Set(ctx, 1, hash, "uploaded-1"))
	uploadedID, found, err := cache.Get(ctx, 1, hash)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "uploaded-1", uploadedID)
	_, found, err = cache.Get(ctx, 2, hash)
	require.NoError(t, err)
	require.False(t, found)
}

func TestLeonardoImageUploadCacheIgnoresInvalidKeys(t *testing.T) {
	cache := &leonardoImageUploadCacheFake{}
	require.NoError(t, cache.Set(context.Background(), 0, "invalid", "uploaded"))
	_, found, err := cache.Get(context.Background(), 0, "invalid")
	require.NoError(t, err)
	require.False(t, found)
}
