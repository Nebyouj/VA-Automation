'use client';

import { useEffect, useState, useCallback, use } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowLeft, RefreshCw, XCircle, Clock, Calendar, CheckCircle2 } from 'lucide-react';
import { getJob, cancelJob } from '@/lib/api';
import { Job } from '@/lib/types';
import { formatDate } from '@/lib/utils';
import { JobStatusBadge } from '@/components/jobs/job-status';
import { JobPriority } from '@/components/jobs/job-priority';
import { JobPayload } from '@/components/jobs/job-payload';
import { useToast } from '@/components/ui/toast';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { ErrorState } from '@/components/ui/error-state';
import { Skeleton, SkeletonText } from '@/components/ui/skeleton';

export default function JobDetailsPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const id = parseInt(resolvedParams.id, 10);
  
  const router = useRouter();
  const { toast } = useToast();
  
  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  
  const [cancelling, setCancelling] = useState(false);
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);

  const fetchJob = useCallback(async (showRefresh = false) => {
    try {
      if (showRefresh) setRefreshing(true);
      const data = await getJob(id);
      setJob(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load job');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [id]);

  useEffect(() => {
    if (isNaN(id)) {
      setError('Invalid job ID');
      setLoading(false);
      return;
    }
    
    fetchJob();
    
    // Poll if job is pending or running
    let interval: NodeJS.Timeout;
    if (job?.status === 'Pending' || job?.status === 'running') {
      interval = setInterval(() => fetchJob(), 5000);
    }
    
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [id, fetchJob, job?.status]);

  const handleCancel = async () => {
    setCancelling(true);
    try {
      await cancelJob(id);
      toast('success', 'Job cancelled successfully');
      fetchJob();
    } catch (err) {
      toast('error', err instanceof Error ? err.message : 'Failed to cancel job');
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6 max-w-4xl animate-in fade-in duration-300">
        <div className="flex items-center gap-4">
          <Skeleton className="w-8 h-8 rounded-full" />
          <Skeleton className="w-32 h-8" />
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="md:col-span-2 space-y-6">
            <Skeleton className="h-64 w-full rounded-xl" />
          </div>
          <div className="space-y-6">
            <Skeleton className="h-48 w-full rounded-xl" />
            <Skeleton className="h-32 w-full rounded-xl" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !job) {
    return (
      <div className="max-w-4xl space-y-6">
        <Link
          href="/jobs"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-200 transition-colors"
        >
          <ArrowLeft size={16} />
          Back to jobs
        </Link>
        <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50">
          <ErrorState description={error || 'Job not found'} onRetry={() => fetchJob()} />
        </div>
      </div>
    );
  }

  const canCancel = job.status === 'Pending' || job.status === 'running';

  return (
    <div className="max-w-4xl space-y-6 animate-in fade-in duration-300">
      <ConfirmDialog
        open={showCancelConfirm}
        title="Cancel Job"
        description="Are you sure you want to cancel this job? This action cannot be undone."
        confirmLabel="Cancel Job"
        variant="danger"
        onConfirm={handleCancel}
        onCancel={() => setShowCancelConfirm(false)}
      />

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-3">
          <Link
            href="/jobs"
            className="p-1.5 rounded-lg text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
            aria-label="Back to jobs"
          >
            <ArrowLeft size={18} />
          </Link>
          <h1 className="text-xl sm:text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100 flex items-center gap-3">
            <span className="text-neutral-400 dark:text-neutral-500 font-mono text-lg sm:text-xl">
              #{job.id}
            </span>
            {job.type}
          </h1>
        </div>

        <div className="flex items-center gap-2 pl-9 sm:pl-0">
          <button
            onClick={() => fetchJob(true)}
            disabled={refreshing}
            className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium text-neutral-600 dark:text-neutral-400 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors disabled:opacity-50"
          >
            <RefreshCw size={13} className={refreshing ? 'animate-spin' : ''} />
            Refresh
          </button>
          
          {canCancel && (
            <button
              onClick={() => setShowCancelConfirm(true)}
              disabled={cancelling}
              className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium text-red-600 dark:text-red-400 bg-white dark:bg-neutral-900 border border-red-200 dark:border-red-900/50 rounded-lg hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors disabled:opacity-50"
            >
              <XCircle size={13} />
              Cancel Job
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Main content - 2 columns */}
        <div className="md:col-span-2 space-y-6">
          <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 dark:border-neutral-800">
              <h2 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                Payload
              </h2>
            </div>
            <div className="p-5">
              <JobPayload payload={job.payload} />
            </div>
          </div>
        </div>

        {/* Sidebar - 1 column */}
        <div className="space-y-6">
          {/* Status & Meta */}
          <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 dark:border-neutral-800">
              <h2 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                Details
              </h2>
            </div>
            <div className="p-5 space-y-5">
              <div>
                <span className="block text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider mb-2">
                  Status
                </span>
                <JobStatusBadge status={job.status} size="md" />
              </div>
              
              <div>
                <span className="block text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider mb-2">
                  Priority
                </span>
                <JobPriority priority={job.priority} />
              </div>

              <div>
                <span className="block text-xs font-medium text-neutral-500 dark:text-neutral-400 uppercase tracking-wider mb-2">
                  Attempts
                </span>
                <span className="text-sm font-medium tabular-nums text-neutral-900 dark:text-neutral-100">
                  {job.attempts} <span className="text-neutral-400">/ {job.max_attempts}</span>
                </span>
              </div>
            </div>
          </div>

          {/* Timeline */}
          <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 overflow-hidden">
            <div className="px-5 py-4 border-b border-neutral-100 dark:border-neutral-800">
              <h2 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                Timeline
              </h2>
            </div>
            <div className="p-5 space-y-4">
              <TimelineItem 
                icon={<Calendar size={14} />} 
                label="Created" 
                value={formatDate(job.created_at)} 
              />
              <TimelineItem 
                icon={<Clock size={14} />} 
                label="Started" 
                value={formatDate(job.started_at)} 
              />
              <TimelineItem 
                icon={<CheckCircle2 size={14} />} 
                label="Ended" 
                value={formatDate(job.ended_at)} 
                isLast
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function TimelineItem({ 
  icon, 
  label, 
  value, 
  isLast = false 
}: { 
  icon: React.ReactNode; 
  label: string; 
  value: string; 
  isLast?: boolean 
}) {
  const isPending = value === '—';
  
  return (
    <div className="relative flex gap-4">
      {!isLast && (
        <div className="absolute left-3.5 top-7 bottom-0 w-px bg-neutral-200 dark:bg-neutral-800 -translate-x-1/2" />
      )}
      <div className={`relative z-10 w-7 h-7 rounded-full border flex items-center justify-center shrink-0 ${
        isPending 
          ? 'bg-neutral-50 dark:bg-neutral-900 border-neutral-200 dark:border-neutral-800 text-neutral-400 dark:text-neutral-500' 
          : 'bg-white dark:bg-neutral-950 border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300'
      }`}>
        {icon}
      </div>
      <div className="pt-1 pb-4">
        <p className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
          {label}
        </p>
        <p className={`mt-0.5 text-sm ${isPending ? 'text-neutral-400 dark:text-neutral-500' : 'text-neutral-900 dark:text-neutral-100 font-medium tabular-nums'}`}>
          {value}
        </p>
      </div>
    </div>
  );
}
