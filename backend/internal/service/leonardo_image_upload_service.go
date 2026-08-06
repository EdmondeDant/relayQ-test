package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
)

var ErrLeonardoImageUploadInvalid = errors.New("leonardo image upload is invalid")

type LeonardoInitImageClient interface {
	CreateInitImageUpload(context.Context, string) (*leonardo.InitImageUpload, error)
	UploadInitImage(context.Context, *leonardo.InitImageUpload, string, []byte) error
}

type LeonardoImageUploadService struct {
	cache *LeonardoImageUploadCache
}

func NewLeonardoImageUploadService(cache *LeonardoImageUploadCache) *LeonardoImageUploadService {
	return &LeonardoImageUploadService{cache: cache}
}

func (s *LeonardoImageUploadService) Upload(ctx context.Context, accountID int64, client LeonardoInitImageClient, input *LeonardoImageInput) (string, error) {
	if s == nil || accountID <= 0 || client == nil || input == nil || len(input.Data) == 0 || strings.TrimSpace(input.Extension) == "" {
		return "", ErrLeonardoImageUploadInvalid
	}
	hash := LeonardoImageSHA256(input.Data)
	if uploadedID, found, err := s.cache.Get(ctx, accountID, hash); err == nil && found {
		return uploadedID, nil
	}
	presigned, err := client.CreateInitImageUpload(ctx, input.Extension)
	if err != nil {
		return "", err
	}
	if presigned == nil || strings.TrimSpace(presigned.ID) == "" {
		return "", ErrLeonardoImageUploadInvalid
	}
	filename := strings.TrimSpace(input.FileName)
	if filename == "" {
		filename = "image." + input.Extension
	}
	if err := client.UploadInitImage(ctx, presigned, filename, input.Data); err != nil {
		return "", err
	}
	_ = s.cache.Set(ctx, accountID, hash, presigned.ID)
	return presigned.ID, nil
}
