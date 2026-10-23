import { Job, JobStats, CreateJobRequest } from './types';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${API_URL}${path}`;

  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => 'Unknown error');
    throw new ApiError(text.trim() || `Request failed with status ${res.status}`, res.status);
  }

  const text = await res.text();
  if (!text) return null as T;

  return JSON.parse(text) as T;
}

// --- Jobs ---

export async function getJobs(params?: {
  status?: string;
  type?: string;
  page?: number;
  limit?: number;
}): Promise<Job[]> {
  const searchParams = new URLSearchParams();
  if (params?.status) searchParams.set('status', params.status);
  if (params?.type) searchParams.set('type', params.type);
  if (params?.page) searchParams.set('page', String(params.page));
  if (params?.limit) searchParams.set('limit', String(params.limit));

  const query = searchParams.toString();
  const path = query ? `/jobs?${query}` : '/jobs';

  const jobs = await request<Job[] | null>(path);
  return jobs ?? [];
}

export async function getJob(id: number): Promise<Job> {
  return request<Job>(`/jobs/${id}`);
}

export async function createJob(data: CreateJobRequest): Promise<Job> {
  return request<Job>('/jobs', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function cancelJob(id: number): Promise<Job> {
  return request<Job>(`/jobs/${id}`, {
    method: 'POST',
  });
}

// --- Stats ---

export async function getStats(): Promise<JobStats> {
  return request<JobStats>('/stats');
}

// --- Health check ---

export async function checkHealth(): Promise<boolean> {
  try {
    await request<JobStats>('/stats');
    return true;
  } catch {
    return false;
  }
}

export { ApiError };
