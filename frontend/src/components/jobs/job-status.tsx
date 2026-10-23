import { getStatusMeta } from '@/lib/utils';
import { JobStatus } from '@/lib/types';

interface JobStatusBadgeProps {
  status: JobStatus;
  size?: 'sm' | 'md';
}

export function JobStatusBadge({ status, size = 'sm' }: JobStatusBadgeProps) {
  const meta = getStatusMeta(status);

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full font-medium ${meta.badgeClass} ${
        size === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-2.5 py-1 text-xs'
      }`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${meta.dotClass}`} />
      {meta.label}
    </span>
  );
}
