package api

import (
	"testing"
)

func TestResolveRepo(t *testing.T) {
	tests := []struct {
		name          string
		repoFlag      string
		expectedHost  string
		expectedOwner string
		expectedName  string
		expectErr     bool
	}{
		{
			name:          "standard owner/repo",
			repoFlag:      "cli/cli",
			expectedOwner: "cli",
			expectedName:  "cli",
			expectErr:     false,
		},
		{
			name:          "host qualified repository",
			repoFlag:      "github.com/golang/go",
			expectedHost:  "github.com",
			expectedOwner: "golang",
			expectedName:  "go",
			expectErr:     false,
		},
		{
			name:          "enterprise host qualified repository",
			repoFlag:      "github.mycompany.com/team/repo",
			expectedHost:  "github.mycompany.com",
			expectedOwner: "team",
			expectedName:  "repo",
			expectErr:     false,
		},
		{
			name:      "invalid repository format without slash",
			repoFlag:  "invalidformatwithoutslash",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ResolveRepo(tc.repoFlag)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for repo %q, got nil", tc.repoFlag)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for repo %q: %v", tc.repoFlag, err)
			}

			if r.Owner != tc.expectedOwner || r.Name != tc.expectedName {
				t.Errorf("expected %s/%s, got %s/%s", tc.expectedOwner, tc.expectedName, r.Owner, r.Name)
			}

			if tc.expectedHost != "" && r.Host != tc.expectedHost {
				t.Errorf("expected host %s, got %s", tc.expectedHost, r.Host)
			}
		})
	}
}
