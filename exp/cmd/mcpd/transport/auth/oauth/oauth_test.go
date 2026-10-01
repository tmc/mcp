package oauth

import (
	"testing"

	"github.com/tmc/mcp/exp/cmd/mcpd/transport/auth/authtypes"
)

func TestCreateOAuthConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *authtypes.Config
		wantURL string
		wantErr bool
	}{
		{
			name:    "github",
			config:  &authtypes.Config{Provider: "github"},
			wantURL: "https://github.com/login/oauth/authorize",
		},
		{
			name: "custom",
			config: &authtypes.Config{
				Provider:    "custom",
				AuthURL:     "https://auth.example/authorize",
				TokenURL:    "https://auth.example/token",
				UserInfoURL: "https://auth.example/userinfo",
			},
			wantURL: "https://auth.example/authorize",
		},
		{
			name:    "google removed",
			config:  &authtypes.Config{Provider: "google"},
			wantErr: true,
		},
		{
			name:    "custom requires endpoints",
			config:  &authtypes.Config{Provider: "custom"},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := createOAuthConfig(test.config)
			if (err != nil) != test.wantErr {
				t.Fatalf("createOAuthConfig() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got.Endpoint.AuthURL != test.wantURL {
				t.Errorf("AuthURL = %q, want %q", got.Endpoint.AuthURL, test.wantURL)
			}
		})
	}
}
