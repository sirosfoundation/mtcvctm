package cmd

import (
	"strings"
	"testing"
)

func TestGenerateSchemaMetaScaffold(t *testing.T) {
	tests := []struct {
		name           string
		credName       string
		generatedFiles []string
		wantContains   []string
	}{
		{
			name:           "with credential name",
			credName:       "Identity Credential",
			generatedFiles: []string{"identity.vctm.json"},
			wantContains: []string{
				"attestation_los: low",
				"binding_type: cnf",
				"# Credential: Identity Credential",
			},
		},
		{
			name:           "without credential name",
			credName:       "",
			generatedFiles: nil,
			wantContains: []string{
				"attestation_los: low",
				"binding_type: cnf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSchemaMetaScaffold(tt.credName, tt.generatedFiles)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("scaffold missing %q, got:\n%s", want, result)
				}
			}
		})
	}
}
