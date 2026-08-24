package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenAIImagesRequestFiltersClientFingerprintHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Request.Header.Set("Accept", "application/json")
	c.Request.Header.Set("Accept-Language", "zh-CN")
	c.Request.Header.Set("User-Agent", "OpenAI/Python 2.0.0")
	c.Request.Header.Set("OpenAI-Beta", "responses=v1")
	c.Request.Header.Set("X-Stainless-Lang", "python")
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.10")

	account := newOpenAIImageGenerationControlTestAccount()
	service := newOpenAIImageGenerationControlTestService(nil)
	req, err := service.buildOpenAIImagesRequest(
		context.Background(), c, account, []byte(`{"model":"Nano Banana"}`),
		"application/json", "sk-upstream", openAIImagesGenerationsEndpoint,
	)
	require.NoError(t, err)
	require.Equal(t, "Bearer sk-upstream", req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Accept"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Equal(t, openaiImagesDefaultUserAgent, req.Header.Get("User-Agent"))
	require.Empty(t, req.Header.Get("Accept-Language"))
	require.Empty(t, req.Header.Get("OpenAI-Beta"))
	require.Empty(t, req.Header.Get("X-Stainless-Lang"))
	require.Empty(t, req.Header.Get("X-Forwarded-For"))
}

func TestBuildOpenAIImagesRequestUsesConfiguredUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Request.Header.Set("User-Agent", "blocked-client-agent")

	account := newOpenAIImageGenerationControlTestAccount()
	account.Credentials["user_agent"] = "Trusted-Upstream-Agent/1.0"
	service := newOpenAIImageGenerationControlTestService(nil)
	req, err := service.buildOpenAIImagesRequest(
		context.Background(), c, account, nil, "application/json", "sk-upstream", openAIImagesGenerationsEndpoint,
	)
	require.NoError(t, err)
	require.Equal(t, "Trusted-Upstream-Agent/1.0", req.Header.Get("User-Agent"))
}
