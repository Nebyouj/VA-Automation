// API response types matching the Go backend exactly

export type JobStatus = 'Pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface Job {
  id: number;
  type: string;
  status: JobStatus;
  priority: number;
  payload?: Record<string, unknown>;
  attempts: number;
  max_attempts: number;
  created_at: string;
  started_at: string | null;
  ended_at: string | null;
}

export interface JobStats {
  pending: number;
  running: number;
  completed: number;
  failed: number;
  cancelled: number;
}

export interface CreateJobRequest {
  type: string;
  priority: number;
  payload: Record<string, unknown>;
}

export type JobType = 'send_email' | 'generate_report' | 'process_file' | 'failing_job';

export const JOB_TYPES: { value: JobType; label: string; description: string }[] = [
  { value: 'send_email', label: 'Send Email', description: 'Send an email to a recipient' },
  { value: 'generate_report', label: 'Generate Report', description: 'Generate a report for a user' },
  { value: 'process_file', label: 'Process File', description: 'Process a file with a given operation' },
  { value: 'failing_job', label: 'Failing Job', description: 'A job that always fails (for testing)' },
];
