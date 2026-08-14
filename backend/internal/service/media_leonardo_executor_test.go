package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
	"github.com/stretchr/testify/require"
)

type mediaLeonardoAccountRepo struct {
	AccountRepository
	account *Account
}

func (r *mediaLeonardoAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func TestDefaultMediaLeonardoExecutorSubmitImageAndVideo(t *testing.T) {
	groupID := int64(5)
	account := createLeonardoAccount()
	account.GroupIDs = []int64{groupID}
	tests := []struct {
		name       string
		request    MediaCanonicalRequest
		model      string
		parameters map[string]any
	}{
		{name: "image", model: "flux-schnell", request: MediaCanonicalRequest{Operation: "generations", Modality: "image", Fields: map[string]any{"prompt": "cat", "size": "896x896", "n": float64(1)}}, parameters: map[string]any{"width": float64(896), "height": float64(896), "quantity": float64(1)}},
		{name: "video", model: "seedance-1.0-pro-fast", request: MediaCanonicalRequest{Operation: "generations", Modality: "video", Fields: map[string]any{"prompt": "cat", "size": "864x480", "duration": float64(4), "n": float64(1)}}, parameters: map[string]any{"duration": float64(4), "quantity": float64(1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			account.Credentials = map[string]any{"model_mapping": map[string]any{test.model: test.model}}
			client := &leonardoGenerationClientMock{response: &leonardo.CreateGenerationResponse{GenerationID: "1dd50843-d653-4516-a8e3-f0238ee453ff"}}
			factory := &leonardoImageCreateClientFactoryFake{client: client}
			create := NewLeonardoMediaCreateService(&mediaLeonardoAccountRepo{account: account}, &LeonardoImageCreateOrchestrator{clients: factory})
			executor := NewDefaultMediaLeonardoExecutor(create, nil)
			job := &GenerationJob{PublicID: "media_rq_1", GroupID: &groupID, AccountID: account.ID}

			outcome, err := executor.SubmitMedia(context.Background(), job, test.request, MediaCatalogOffer{Provider: PlatformLeonardo, SourceGroupID: groupID, UpstreamModel: test.model})

			require.NoError(t, err)
			require.Equal(t, MediaSubmissionSubmitted, outcome.State)
			require.Equal(t, "1dd50843-d653-4516-a8e3-f0238ee453ff", outcome.UpstreamID)
			require.Equal(t, 1, client.calls)
			var body map[string]any
			require.NoError(t, json.Unmarshal(client.rawRequest, &body))
			require.Equal(t, test.model, body["model"])
			parameters := body["parameters"].(map[string]any)
			for key, value := range test.parameters {
				require.Equal(t, value, parameters[key])
			}
		})
	}
}

func TestDefaultMediaLeonardoExecutorSubmissionSemantics(t *testing.T) {
	groupID := int64(5)
	account := createLeonardoAccount()
	account.GroupIDs = []int64{groupID}
	account.Credentials = map[string]any{"model_mapping": map[string]any{"flux-schnell": "flux-schnell"}}
	tests := []struct {
		name  string
		err   error
		state MediaSubmissionState
	}{
		{name: "not written", err: ErrLeonardoGenerationRequestNotWritten, state: MediaSubmissionNotWritten},
		{name: "unknown", err: errors.New("connection reset after write"), state: MediaSubmissionSideEffectUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &leonardoGenerationClientMock{err: test.err}
			create := NewLeonardoMediaCreateService(&mediaLeonardoAccountRepo{account: account}, &LeonardoImageCreateOrchestrator{clients: &leonardoImageCreateClientFactoryFake{client: client}})
			outcome, err := NewDefaultMediaLeonardoExecutor(create, nil).SubmitMedia(context.Background(), &GenerationJob{GroupID: &groupID, AccountID: account.ID}, MediaCanonicalRequest{Operation: "generations", Modality: "image", Fields: map[string]any{"prompt": "cat", "size": "896x896"}}, MediaCatalogOffer{SourceGroupID: groupID, UpstreamModel: "flux-schnell"})
			require.Error(t, err)
			require.Equal(t, test.state, outcome.State)
			require.Equal(t, 1, client.calls)
			if test.state == MediaSubmissionSideEffectUnknown {
				require.Contains(t, outcome.Result, "submission_diagnostic")
			}
		})
	}
}
