package job

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		job         Job
		wantErr     bool
		wantInvalid bool
	}{
		{
			name: "valid send email",
			job: Job{
				Type:    "send_email",
				Payload: map[string]any{"to": "test@example.com"},
			},
		},
		{
			name: "send email missing recipient",
			job: Job{
				Type:    "send_email",
				Payload: map[string]any{},
			},
			wantErr: true,
		},
		{
			name: "valid generate report",
			job: Job{
				Type: "generate_report",
				Payload: map[string]any{
					"report_type": "daily",
					"user_id":     float64(1),
				},
			},
		},
		{
			name: "generate report missing report type",
			job: Job{
				Type: "generate_report",
				Payload: map[string]any{
					"user_id": float64(1),
				},
			},
			wantErr: true,
		},
		{
			name: "generate report missing user ID",
			job: Job{
				Type: "generate_report",
				Payload: map[string]any{
					"report_type": "daily",
				},
			},
			wantErr: true,
		},
		{
			name: "generate report with invalid user ID",
			job: Job{
				Type: "generate_report",
				Payload: map[string]any{
					"report_type": "daily",
					"user_id":     float64(0),
				},
			},
			wantErr: true,
		},
		{
			name: "valid process file",
			job: Job{
				Type:    "process_file",
				Payload: map[string]any{"file_name": "report.csv", "operation": "read_text"},
			},
		},
		{
			name: "process file missing file name",
			job: Job{
				Type:    "process_file",
				Payload: map[string]any{},
			},
			wantErr:     true,
			wantInvalid: true,
		},
		{
			name: "valid failing job",
			job: Job{
				Type:    "failing_job",
				Payload: map[string]any{},
			},
		},
		{
			name: "unknown job type",
			job: Job{
				Type:    "unknown_job",
				Payload: map[string]any{},
			},
			wantErr: true,
		},
		{
			name: "valid cron expression",
			job: Job{
				Type:           "failing_job",
				CronExpression: "*/2 * * * *",
			},
		},
		{
			name: "valid cron alias",
			job: Job{
				Type:           "failing_job",
				CronExpression: "@daily",
			},
		},
		{
			name: "invalid cron expression",
			job: Job{
				Type:           "failing_job",
				CronExpression: "invalid-cron",
			},
			wantErr:     true,
			wantInvalid: true,
		},
		{
			name: "negative repeat interval",
			job: Job{
				Type:               "failing_job",
				RepeatEverySeconds: -1,
			},
			wantErr:     true,
			wantInvalid: true,
		},
		{
			name: "cron and interval both configured",
			job: Job{
				Type:               "failing_job",
				RepeatEverySeconds: 60,
				CronExpression:     "*/2 * * * *",
			},
			wantErr:     true,
			wantInvalid: true,
		},
		{
			name: "zero repeat interval is allowed",
			job: Job{
				Type:               "failing_job",
				RepeatEverySeconds: 0,
			},
		},
		{
			name: "whitespace cron expression is ignored",
			job: Job{
				Type:           "failing_job",
				CronExpression: "   ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.job)

			if tt.wantErr && err == nil {
				t.Fatal("Validate() expected an error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}

			if tt.wantInvalid && !errors.Is(err, ErrInvalidJob) {
				t.Errorf(
					"Validate() error = %v, want an error wrapping ErrInvalidJob",
					err,
				)
			}
		})
	}
}
