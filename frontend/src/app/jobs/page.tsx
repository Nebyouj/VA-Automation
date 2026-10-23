'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import { Search, Filter, RefreshCw, ChevronLeft, ChevronRight, X } from 'lucide-react';
import { getJobs } from '@/lib/api';
import { Job, JobStatus } from '@/lib/types';
import { timeAgo, summarizePayload } from '@/lib/utils';
import { JobStatusBadge } from '@/components/jobs/job-status';
import { JobPriority } from '@/components/jobs/job-priority';
import { EmptyState } from '@/components/ui/empty-state';
import { ErrorState } from '@/components/ui/error-state';
import { JobRowSkeleton } from '@/components/ui/skeleton';

const STATUS_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'All statuses' },
  { value: 'Pending', label: 'Pending' },
  { value: 'running', label: 'Running' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
  { value: 'cancelled', label: 'Cancelled' },
];

const TYPE_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'All types' },
  { value: 'send_email', label: 'Send Email' },
  { value: 'generate_report', label: 'Generate Report' },
  { value: 'process_file', label: 'Process File' },
  { value: 'failing_job', label: 'Failing Job' },
];

const PAGE_SIZE = 15;

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  // Filters
  const [statusFilter, setStatusFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [page, setPage] = useState(1);

  const fetchJobs = useCallback(
    async (showRefresh = false) => {
      try {
        if (showRefresh) setRefreshing(true);
        else setLoading(true);
        const data = await getJobs({
          status: statusFilter || undefined,
          type: typeFilter || undefined,
          page,
          limit: PAGE_SIZE,
        });
        setJobs(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load jobs');
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [statusFilter, typeFilter, page]
  );

  useEffect(() => {
    fetchJobs();
  }, [fetchJobs]);

  const clearFilters = () => {
    setStatusFilter('');
    setTypeFilter('');
    setPage(1);
  };

  const hasFilters = statusFilter || typeFilter;

  return (
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
            Jobs
          </h1>
          <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
            Browse and manage all queued jobs
          </p>
        </div>
        <button
          onClick={() => fetchJobs(true)}
          disabled={refreshing}
          className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium text-neutral-600 dark:text-neutral-400 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors disabled:opacity-50"
        >
          <RefreshCw size={13} className={refreshing ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative">
          <Filter size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400" />
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="appearance-none pl-8 pr-8 py-2 text-xs font-medium rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 text-neutral-700 dark:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 cursor-pointer"
          >
            {STATUS_FILTERS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </select>
        </div>

        <select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value);
            setPage(1);
          }}
          className="appearance-none px-3 py-2 text-xs font-medium rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 text-neutral-700 dark:text-neutral-300 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 cursor-pointer"
        >
          {TYPE_FILTERS.map((f) => (
            <option key={f.value} value={f.value}>
              {f.label}
            </option>
          ))}
        </select>

        {hasFilters && (
          <button
            onClick={clearFilters}
            className="inline-flex items-center gap-1 px-2.5 py-2 text-xs font-medium text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200 transition-colors"
          >
            <X size={12} />
            Clear
          </button>
        )}
      </div>

      {/* Job List */}
      <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 overflow-hidden">
        {/* Table header */}
        <div className="hidden sm:grid grid-cols-[3rem_1fr_4rem_5.5rem_4rem_5rem] gap-3 px-5 py-2.5 text-[11px] font-medium text-neutral-400 dark:text-neutral-500 uppercase tracking-wider border-b border-neutral-100 dark:border-neutral-800">
          <span>ID</span>
          <span>Type</span>
          <span>Pri</span>
          <span>Status</span>
          <span>Tries</span>
          <span className="text-right">Time</span>
        </div>

        {loading ? (
          <div>
            {Array.from({ length: 8 }).map((_, i) => (
              <JobRowSkeleton key={i} />
            ))}
          </div>
        ) : error ? (
          <ErrorState description={error} onRetry={() => fetchJobs()} />
        ) : jobs.length === 0 ? (
          <EmptyState
            title={hasFilters ? 'No matching jobs' : 'No jobs yet'}
            description={
              hasFilters
                ? "Try adjusting your filters to find what you're looking for."
                : "Jobs will appear here once they are created."
            }
            action={
              hasFilters ? (
                <button
                  onClick={clearFilters}
                  className="text-xs font-medium text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-200 transition-colors"
                >
                  Clear filters
                </button>
              ) : undefined
            }
          />
        ) : (
          <div className="divide-y divide-neutral-100 dark:divide-neutral-800">
            {jobs.map((job) => (
              <Link
                key={job.id}
                href={`/jobs/${job.id}`}
                className="grid grid-cols-1 sm:grid-cols-[3rem_1fr_4rem_5.5rem_4rem_5rem] gap-2 sm:gap-3 items-center px-5 py-3.5 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors group"
              >
                {/* Mobile: compact card layout */}
                <div className="sm:hidden flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-xs font-mono text-neutral-400">#{job.id}</span>
                    <span className="text-sm font-medium text-neutral-800 dark:text-neutral-200 truncate">
                      {job.type}
                    </span>
                  </div>
                  <JobStatusBadge status={job.status} />
                </div>
                <div className="sm:hidden flex items-center gap-3 text-xs text-neutral-500 dark:text-neutral-400">
                  <JobPriority priority={job.priority} />
                  <span>{job.attempts} attempt{job.attempts !== 1 ? 's' : ''}</span>
                  <span className="tabular-nums">{timeAgo(job.created_at)}</span>
                </div>

                {/* Desktop: grid row */}
                <span className="hidden sm:inline text-xs font-mono text-neutral-400 dark:text-neutral-500">
                  #{job.id}
                </span>
                <div className="hidden sm:flex items-center gap-2 min-w-0">
                  <span className="text-sm font-medium text-neutral-800 dark:text-neutral-200 truncate">
                    {job.type}
                  </span>
                  <span className="text-xs text-neutral-400 dark:text-neutral-500 truncate hidden lg:inline">
                    {summarizePayload(job.payload)}
                  </span>
                </div>
                <span className="hidden sm:inline">
                  <JobPriority priority={job.priority} />
                </span>
                <span className="hidden sm:inline">
                  <JobStatusBadge status={job.status} />
                </span>
                <span className="hidden sm:inline text-xs text-neutral-500 dark:text-neutral-400 tabular-nums">
                  {job.attempts}/{job.max_attempts}
                </span>
                <span className="hidden sm:inline text-xs text-neutral-400 dark:text-neutral-500 text-right tabular-nums">
                  {timeAgo(job.created_at)}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {!loading && !error && jobs.length > 0 && (
        <div className="flex items-center justify-between">
          <span className="text-xs text-neutral-400 dark:text-neutral-500">
            Page {page}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
              className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronLeft size={13} />
              Prev
            </button>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={jobs.length < PAGE_SIZE}
              className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-neutral-600 dark:text-neutral-400 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
            >
              Next
              <ChevronRight size={13} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
