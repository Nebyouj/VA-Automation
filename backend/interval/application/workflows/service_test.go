package workflows

import (
	"strings"
	"testing"
)

func TestValidateWorkflow(t *testing.T) {
	tests := []struct {
		name    string
		request CreateRequest
		wantErr string
	}{
		{
			name: "valid workflow with dependencies",
			request: CreateRequest{
				Name: "Report workflow",
				Jobs: []JobDefinition{
					{
						Key:  "read",
						Type: "failing_job",
					},
					{
						Key:       "report",
						Type:      "failing_job",
						DependsOn: []string{"read"},
					},
					{
						Key:       "email",
						Type:      "failing_job",
						DependsOn: []string{"report"},
					},
				},
			},
		},
		{
			name: "missing workflow name",
			request: CreateRequest{
				Name: " ",
				Jobs: []JobDefinition{
					{Key: "first", Type: "failing_job"},
				},
			},
			wantErr: "workflow name is required",
		},
		{
			name: "workflow without jobs",
			request: CreateRequest{
				Name: "Empty workflow",
			},
			wantErr: "at least one job",
		},
		{
			name: "missing job key",
			request: CreateRequest{
				Name: "Missing key",
				Jobs: []JobDefinition{
					{Type: "failing_job"},
				},
			},
			wantErr: "requires a key",
		},
		{
			name: "duplicate job keys",
			request: CreateRequest{
				Name: "Duplicate keys",
				Jobs: []JobDefinition{
					{Key: "first", Type: "failing_job"},
					{Key: "first", Type: "failing_job"},
				},
			},
			wantErr: "duplicate workflow job key",
		},
		{
			name: "unknown dependency",
			request: CreateRequest{
				Name: "Unknown dependency",
				Jobs: []JobDefinition{
					{
						Key:       "first",
						Type:      "failing_job",
						DependsOn: []string{"missing"},
					},
				},
			},
			wantErr: "unknown job key",
		},
		{
			name: "job depends on itself",
			request: CreateRequest{
				Name: "Self dependency",
				Jobs: []JobDefinition{
					{
						Key:       "first",
						Type:      "failing_job",
						DependsOn: []string{"first"},
					},
				},
			},
			wantErr: "cannot depend on itself",
		},
		{
			name: "duplicate dependencies",
			request: CreateRequest{
				Name: "Duplicate dependencies",
				Jobs: []JobDefinition{
					{Key: "first", Type: "failing_job"},
					{
						Key:       "second",
						Type:      "failing_job",
						DependsOn: []string{"first", "first"},
					},
				},
			},
			wantErr: "duplicate dependency",
		},
		{
			name: "dependency cycle",
			request: CreateRequest{
				Name: "Cyclic workflow",
				Jobs: []JobDefinition{
					{
						Key:       "first",
						Type:      "failing_job",
						DependsOn: []string{"second"},
					},
					{
						Key:       "second",
						Type:      "failing_job",
						DependsOn: []string{"first"},
					},
				},
			},
			wantErr: "dependency cycle",
		},
		{
			name: "recurring job rejected",
			request: CreateRequest{
				Name: "Recurring workflow",
				Jobs: []JobDefinition{
					{
						Key:                "first",
						Type:               "failing_job",
						RepeatEverySeconds: 60,
					},
				},
			},
			wantErr: "not supported in workflows",
		},
		{
			name: "unknown job type",
			request: CreateRequest{
				Name: "Unknown job",
				Jobs: []JobDefinition{
					{Key: "first", Type: "does_not_exist"},
				},
			},
			wantErr: "unknown job type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.request)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() returned unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validate() expected error containing %q, got nil", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf(
					"validate() error = %q, want substring %q",
					err.Error(),
					tt.wantErr,
				)
			}
		})
	}
}
