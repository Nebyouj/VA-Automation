'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import { PlusCircle, ArrowRight, RefreshCw, Activity, Clock, Zap } from 'lucide-react';
import { getStats, getJobs } from '@/lib/api';
import { Job, JobStats } from '@/lib/types';
import { formatNumber, timeAgo, summarizePayload } from '@/lib/utils';
import { JobStatusBadge } from '@/components/jobs/job-status';
import { JobPriority } from '@/components/jobs/job-priority';
import { DashboardSkeleton } from '@/components/ui/skeleton';
import { ErrorState } from '@/components/ui/error-state';
import { EmptyState } from '@/components/ui/empty-state';

export default function DashboardPage() {
  const [stats, setStats] = useState<JobStats | null>(null);
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const fetchData = useCallback(async (showRefresh = false) => {
    try {
      if (showRefresh) setRefreshing(true);
      const [statsData, jobsData] = await Promise.all([
        getStats(),
        getJobs({ limit: 7 }),
      ]);
      setStats(statsData);
      setRecentJobs(jobsData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(() => fetchData(), 15000);
    return () => clearInterval(interval);
  }, [fetchData]);

  if (loading) return <DashboardSkeleton />;
  if (error) return <ErrorState description={error} onRetry={() => fetchData()} />;
  if (!stats) return null;

  const total = stats.pending + stats.running + stats.completed + stats.failed + stats.cancelled;

  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
            Overview
          </h1>
          <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
            Monitor your job queue at a glance
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => fetchData(true)}
            disabled={refreshing}
            className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium text-neutral-600 dark:text-neutral-400 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors disabled:opacity-50"
            aria-label="Refresh data"
          >
            <RefreshCw size={13} className={refreshing ? 'animate-spin' : ''} />
            Refresh
          </button>
          <Link
            href="/create"
            className="inline-flex items-center gap-2 px-4 py-2 text-xs font-medium bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 rounded-lg hover:bg-neutral-800 dark:hover:bg-neutral-100 transition-colors"
          >
            <PlusCircle size={13} />
            New Job
          </Link>
        </div>
      </div>

      {/* Hero stat + secondary stats — asymmetric layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        {/* Large primary stat */}
        <div className="lg:col-span-5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 p-6">
          <div className="flex items-center gap-2 text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">
            <Activity size={13} />
            Total Jobs
          </div>
          <div className="mt-3 text-5xl font-bold tabular-nums tracking-tight text-neutral-900 dark:text-neutral-100">
            {formatNumber(total)}
          </div>
          <div className="mt-3 flex items-center gap-4 text-xs text-neutral-400 dark:text-neutral-500">
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />
              {stats.running} active
            </span>
            <span className="flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-neutral-300 dark:bg-neutral-600" />
              {stats.pending} queued
            </span>
          </div>
        </div>

        {/* Secondary stats in a 2x2 / varied grid */}
        <div className="lg:col-span-7 grid grid-cols-2 sm:grid-cols-4 gap-4">
          <StatCard label="Pending" value={stats.pending} icon={<Clock size={13} />} />
          <StatCard
            label="Running"
            value={stats.running}
            icon={<Zap size={13} />}
            accent
          />
          <StatCard
            label="Completed"
            value={stats.completed}
            className="text-emerald-600 dark:text-emerald-400"
          />
          <StatCard
            label="Failed"
            value={stats.failed}
            className="text-red-500 dark:text-red-400"
          />
        </div>
      </div>

      {/* Cancelled stat (smaller, offset) */}
      <div className="flex justify-end">
        <div className="inline-flex items-center gap-3 px-4 py-2 rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 text-xs text-neutral-500 dark:text-neutral-400">
          <span className="font-medium">Cancelled</span>
          <span className="text-sm font-semibold tabular-nums text-neutral-700 dark:text-neutral-300">
            {formatNumber(stats.cancelled)}
          </span>
        </div>
      </div>

      {/* Main content — asymmetric columns */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        {/* Recent Jobs — wider */}
        <div className="lg:col-span-3">
          <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 overflow-hidden">
            <div className="flex items-center justify-between px-5 py-4 border-b border-neutral-100 dark:border-neutral-800">
              <h2 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                Recent Jobs
              </h2>
              <Link
                href="/jobs"
                className="inline-flex items-center gap-1 text-xs font-medium text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-200 transition-colors"
              >
                View all <ArrowRight size={12} />
              </Link>
            </div>

            {recentJobs.length === 0 ? (
              <EmptyState
                title="No jobs yet"
                description="Create your first job to get started."
                action={
                  <Link
                    href="/create"
                    className="inline-flex items-center gap-2 px-4 py-2 text-xs font-medium bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 rounded-lg hover:bg-neutral-800 dark:hover:bg-neutral-100 transition-colors"
                  >
                    <PlusCircle size={13} />
                    Create Job
                  </Link>
                }
              />
            ) : (
              <div className="divide-y divide-neutral-100 dark:divide-neutral-800">
                {recentJobs.map((job) => (
                  <Link
                    key={job.id}
                    href={`/jobs/${job.id}`}
                    className="flex items-center gap-3 px-5 py-3 hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors group"
                  >
                    <span className="text-xs font-mono text-neutral-400 dark:text-neutral-500 w-8">
                      #{job.id}
                    </span>
                    <span className="text-sm font-medium text-neutral-800 dark:text-neutral-200 truncate flex-1 min-w-0">
                      {job.type}
                    </span>
                    <JobPriority priority={job.priority} />
                    <JobStatusBadge status={job.status} />
                    <span className="text-xs text-neutral-400 dark:text-neutral-500 hidden sm:inline tabular-nums">
                      {timeAgo(job.created_at)}
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right column — queue health + quick info */}
        <div className="lg:col-span-2 space-y-6">
          {/* Queue Health */}
          <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 p-5">
            <h2 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100 mb-4">
              Queue Health
            </h2>
            <div className="space-y-3">
              <HealthRow label="Workers" value="3" sub="configured" />
              <HealthRow
                label="Active"
                value={String(stats.running)}
                sub="processing"
                accent={stats.running > 0}
              />
              <HealthRow label="Queue" value={String(stats.pending)} sub="waiting" />
            </div>

            {/* Visual bar */}
            {total > 0 && (
              <div className="mt-5">
                <div className="flex h-2 rounded-full overflow-hidden bg-neutral-100 dark:bg-neutral-800">
                  {stats.completed > 0 && (
                    <div
                      className="bg-emerald-500 transition-all duration-500"
                      style={{ width: `${(stats.completed / total) * 100}%` }}
                    />
                  )}
                  {stats.running > 0 && (
                    <div
                      className="bg-blue-500 transition-all duration-500"
                      style={{ width: `${(stats.running / total) * 100}%` }}
                    />
                  )}
                  {stats.pending > 0 && (
                    <div
                      className="bg-neutral-300 dark:bg-neutral-600 transition-all duration-500"
                      style={{ width: `${(stats.pending / total) * 100}%` }}
                    />
                  )}
                  {stats.failed > 0 && (
                    <div
                      className="bg-red-500 transition-all duration-500"
                      style={{ width: `${(stats.failed / total) * 100}%` }}
                    />
                  )}
                  {stats.cancelled > 0 && (
                    <div
                      className="bg-amber-400 transition-all duration-500"
                      style={{ width: `${(stats.cancelled / total) * 100}%` }}
                    />
                  )}
                </div>
                <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-neutral-400 dark:text-neutral-500">
                  <span className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" /> Completed
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-blue-500" /> Running
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-neutral-300 dark:bg-neutral-600" /> Pending
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-red-500" /> Failed
                  </span>
                </div>
              </div>
            )}
          </div>

          {/* Quick action card */}
          <Link
            href="/create"
            className="block rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 p-5 hover:border-neutral-300 dark:hover:border-neutral-700 transition-colors group"
          >
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                  Create a Job
                </h3>
                <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                  Send a new task to the queue
                </p>
              </div>
              <div className="w-9 h-9 rounded-lg bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center text-neutral-400 group-hover:text-neutral-600 dark:group-hover:text-neutral-300 transition-colors">
                <PlusCircle size={16} />
              </div>
            </div>
          </Link>
        </div>
      </div>
    </div>
  );
}

// --- Sub-components ---

function StatCard({
  label,
  value,
  icon,
  accent = false,
  className = '',
}: {
  label: string;
  value: number;
  icon?: React.ReactNode;
  accent?: boolean;
  className?: string;
}) {
  return (
    <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 p-4">
      <div className="flex items-center gap-1.5 text-[11px] font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider">
        {icon}
        {label}
      </div>
      <div
        className={`mt-2 text-2xl font-bold tabular-nums tracking-tight ${
          className || (accent ? 'text-blue-600 dark:text-blue-400' : 'text-neutral-900 dark:text-neutral-100')
        }`}
      >
        {formatNumber(value)}
      </div>
    </div>
  );
}

function HealthRow({
  label,
  value,
  sub,
  accent = false,
}: {
  label: string;
  value: string;
  sub: string;
  accent?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-neutral-600 dark:text-neutral-400">{label}</span>
      <div className="flex items-baseline gap-1.5">
        <span
          className={`text-sm font-semibold tabular-nums ${
            accent ? 'text-blue-600 dark:text-blue-400' : 'text-neutral-900 dark:text-neutral-100'
          }`}
        >
          {value}
        </span>
        <span className="text-[11px] text-neutral-400 dark:text-neutral-500">{sub}</span>
      </div>
    </div>
  );
}
