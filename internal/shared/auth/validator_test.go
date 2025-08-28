package auth

import (
	"context"
	"testing"
)

func TestValidate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		token    string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty token",
			token:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "valid spiff_id token",
			token:    "spiff_id_12345",
			expected: "spiff_id_12345",
			wantErr:  false,
		},
		{
			name:     "bearer spiff_id token",
			token:    "Bearer spiff_id_67890",
			expected: "spiff_id_67890",
			wantErr:  false,
		},
		{
			name:     "random token generates fake spiff_id",
			token:    "random_token_123",
			expected: "spiff_id_random_t",
			wantErr:  false,
		},
		{
			name:     "short token generates fake spiff_id",
			token:    "abc",
			expected: "spiff_id_abc",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := Validate(ctx, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
				return
			}

			if userID != tt.expected {
				t.Errorf("Validate() userID = %v, want %v", userID, tt.expected)
			}
		})
	}
}

func TestExtractTokenFromAuthHeader(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "bearer token",
			authHeader: "Bearer spiff_id_12345",
			expected:   "spiff_id_12345",
		},
		{
			name:       "bearer token with spaces",
			authHeader: "Bearer  spiff_id_12345",
			expected:   " spiff_id_12345",
		},
		{
			name:       "token without bearer",
			authHeader: "spiff_id_12345",
			expected:   "spiff_id_12345",
		},
		{
			name:       "lowercase bearer",
			authHeader: "bearer spiff_id_12345",
			expected:   "spiff_id_12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTokenFromAuthHeader(tt.authHeader)
			if result != tt.expected {
				t.Errorf("ExtractTokenFromAuthHeader() = %v, want %v", result, tt.expected)
			}
		})
	}
}
