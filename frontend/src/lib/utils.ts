import { JobStatus } from './types';

/**
 * Format a relative time string like "2m ago", "1h ago", "3d ago"
 */
export function timeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 5) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;

  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

/**
 * Format a date for display
 */
export function formatDate(dateString: string | null): string {
  if (!dateString) return '—';
  const d = new Date(dateString);
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

/**
 * Normalize status display since backend has inconsistent casing
 * (Pending vs running, completed, etc.)
 */
export function normalizeStatus(status: JobStatus): string {
  return status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
}

/**
 * Get status styling metadata
 */
export function getStatusMeta(status: JobStatus): {
  label: string;
  dotClass: string;
  badgeClass: string;
} {
  const normalized = status.toLowerCase();

  switch (normalized) {
    case 'pending':
      return {
        label: 'Pending',
        dotClass: 'bg-neutral-400 dark:bg-neutral-500',
        badgeClass: 'text-neutral-600 dark:text-neutral-400 bg-neutral-100 dark:bg-neutral-800',
      };
    case 'running':
      return {
        label: 'Running',
        dotClass: 'bg-blue-500 animate-pulse',
        badgeClass: 'text-blue-700 dark:text-blue-400 bg-blue-50 dark:bg-blue-950',
      };
    case 'completed':
      return {
        label: 'Completed',
        dotClass: 'bg-emerald-500',
        badgeClass: 'text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950',
      };
    case 'failed':
      return {
        label: 'Failed',
        dotClass: 'bg-red-500',
        badgeClass: 'text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-950',
      };
    case 'cancelled':
      return {
        label: 'Cancelled',
        dotClass: 'bg-amber-400 dark:bg-amber-500',
        badgeClass: 'text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-950',
      };
    default:
      return {
        label: normalized,
        dotClass: 'bg-neutral-400',
        badgeClass: 'text-neutral-600 bg-neutral-100',
      };
  }
}

/**
 * Truncate a string with ellipsis
 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength) + '…';
}

/**
 * Format a number with comma separators
 */
export function formatNumber(n: number): string {
  return n.toLocaleString('en-US');
}

/**
 * Get a summarized payload string for table display
 */
export function summarizePayload(payload?: Record<string, unknown>): string {
  if (!payload || Object.keys(payload).length === 0) return 'No payload';
  const entries = Object.entries(payload);
  const parts = entries.slice(0, 2).map(([k, v]) => `${k}: ${String(v)}`);
  if (entries.length > 2) parts.push(`+${entries.length - 2} more`);
  return parts.join(', ');
}
