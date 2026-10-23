'use client';

import { useEffect, useState } from 'react';
import { RefreshCw, Activity, CheckCircle2, XCircle, Clock, AlertTriangle } from 'lucide-react';
import { getStats } from '@/lib/api';
import { JobStats } from '@/lib/types';
import { formatNumber } from '@/lib/utils';
import { ErrorState } from '@/components/ui/error-state';

export default function StatisticsPage() {
  const [stats, setStats] = useState<JobStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const fetchStats = async (showRefresh = false) => {
    try {
      if (showRefresh) setRefreshing(true);
      const data = await getStats();
      setStats(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load statistics');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchStats();
    const interval = setInterval(() => fetchStats(), 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-8 w-48 bg-neutral-200 dark:bg-neutral-800 rounded-lg" />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-32 rounded-xl bg-neutral-200 dark:bg-neutral-800" />
          ))}
        </div>
      </div>
    );
  }

  if (error || !stats) {
    return <ErrorState description={error || 'Failed to load stats'} onRetry={() => fetchStats()} />;
  }

  const total = stats.pending + stats.running + stats.completed + stats.failed + stats.cancelled;
  const successRate = total > 0 ? ((stats.completed / (stats.completed + stats.failed)) * 100).toFixed(1) : '0.0';

  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
            System Statistics
          </h1>
          <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
            High-level metrics for your job processing pipeline
          </p>
        </div>
        <button
          onClick={() => fetchStats(true)}
          disabled={refreshing}
          className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium text-neutral-600 dark:text-neutral-400 bg-white dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-800 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors disabled:opacity-50"
        >
          <RefreshCw size={13} className={refreshing ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Total Volume */}
        <div className="col-span-1 md:col-span-2 lg:col-span-3 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 p-6 flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2 text-sm font-medium text-neutral-500 dark:text-neutral-400">
              <Activity size={16} />
              Total Jobs Processed
            </div>
            <div className="mt-2 text-4xl font-bold tabular-nums text-neutral-900 dark:text-neutral-100">
              {formatNumber(total)}
            </div>
          </div>
          <div className="text-right">
            <div className="text-sm font-medium text-neutral-500 dark:text-neutral-400">
              Success Rate
            </div>
            <div className={`mt-2 text-3xl font-bold tabular-nums ${
              Number(successRate) > 95 ? 'text-emerald-500' : 'text-amber-500'
            }`}>
              {successRate}%
            </div>
          </div>
        </div>

        {/* Detailed Stats */}
        <StatCard
          title="Completed"
          value={stats.completed}
          total={total}
          icon={<CheckCircle2 size={18} className="text-emerald-500" />}
          colorClass="bg-emerald-500"
        />
        <StatCard
          title="Failed"
          value={stats.failed}
          total={total}
          icon={<XCircle size={18} className="text-red-500" />}
          colorClass="bg-red-500"
        />
        <StatCard
          title="Pending"
          value={stats.pending}
          total={total}
          icon={<Clock size={18} className="text-neutral-400" />}
          colorClass="bg-neutral-400"
        />
        <StatCard
          title="Running"
          value={stats.running}
          total={total}
          icon={<RefreshCw size={18} className="text-blue-500 animate-spin-slow" />}
          colorClass="bg-blue-500"
        />
        <StatCard
          title="Cancelled"
          value={stats.cancelled}
          total={total}
          icon={<AlertTriangle size={18} className="text-amber-500" />}
          colorClass="bg-amber-500"
        />
      </div>
    </div>
  );
}

function StatCard({
  title,
  value,
  total,
  icon,
  colorClass,
}: {
  title: string;
  value: number;
  total: number;
  icon: React.ReactNode;
  colorClass: string;
}) {
  const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0';

  return (
    <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-neutral-600 dark:text-neutral-300">{title}</h3>
        {icon}
      </div>
      <div className="text-3xl font-bold tabular-nums text-neutral-900 dark:text-neutral-100">
        {formatNumber(value)}
      </div>
      <div className="mt-4 flex items-center gap-3">
        <div className="flex-1 h-1.5 rounded-full bg-neutral-100 dark:bg-neutral-800 overflow-hidden">
          <div 
            className={`h-full rounded-full ${colorClass}`} 
            style={{ width: `${percentage}%` }}
          />
        </div>
        <span className="text-xs font-medium text-neutral-500 tabular-nums">
          {percentage}%
        </span>
      </div>
    </div>
  );
}
