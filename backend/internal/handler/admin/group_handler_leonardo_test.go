package admin

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestGroupPlatformBinding(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		target  any
		want    string
		valid   bool
	}{
		{name: "create leonardo", payload: `{"name":"Leonardo","platform":"leonardo"}`, target: &CreateGroupRequest{}, want: "leonardo", valid: true},
		{name: "update leonardo", payload: `{"platform":"leonardo"}`, target: &UpdateGroupRequest{}, want: "leonardo", valid: true},
		{name: "existing platform", payload: `{"name":"Anthropic","platform":"anthropic"}`, target: &CreateGroupRequest{}, want: "anthropic", valid: true},
		{name: "unknown platform", payload: `{"name":"Unknown","platform":"unknown"}`, target: &CreateGroupRequest{}, valid: false},
		{name: "empty create platform", payload: `{"name":"Default"}`, target: &CreateGroupRequest{}, valid: true},
		{name: "empty update platform", payload: `{}`, target: &UpdateGroupRequest{}, valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tt.payload), tt.target); err != nil {
				t.Fatal(err)
			}
			err := binding.Validator.ValidateStruct(tt.target)
			if tt.valid && err != nil {
				t.Fatalf("expected valid binding: %v", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("expected binding validation error")
			}
			if tt.want == "" {
				return
			}
			var got string
			switch request := tt.target.(type) {
			case *CreateGroupRequest:
				got = request.Platform
			case *UpdateGroupRequest:
				got = request.Platform
			}
			if got != tt.want {
				t.Fatalf("platform = %q, want %q", got, tt.want)
			}
		})
	}
}
