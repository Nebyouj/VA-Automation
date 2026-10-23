export function Skeleton({ className = '' }: { className?: string }) {
  return (
    <div
      className={`animate-pulse rounded-lg bg-neutral-200 dark:bg-neutral-800 ${className}`}
    />
  );
}

export function SkeletonText({ lines = 3, className = '' }: { lines?: number; className?: string }) {
  return (
    <div className={`space-y-2.5 ${className}`}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={`h-3.5 ${i === lines - 1 ? 'w-3/4' : 'w-full'}`}
        />
      ))}
    </div>
  );
}

export function StatSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-3 w-20" />
      <Skeleton className="h-9 w-24" />
    </div>
  );
}

export function JobRowSkeleton() {
  return (
    <div className="flex items-center gap-4 px-4 py-3.5 border-b border-neutral-100 dark:border-neutral-800">
      <Skeleton className="h-4 w-10" />
      <Skeleton className="h-4 w-28" />
      <Skeleton className="h-5 w-16 rounded-full" />
      <div className="flex-1" />
      <Skeleton className="h-4 w-14" />
      <Skeleton className="h-4 w-16" />
    </div>
  );
}

export function DashboardSkeleton() {
  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="p-4 rounded-xl border border-neutral-200 dark:border-neutral-800">
            <StatSkeleton />
          </div>
        ))}
      </div>

      {/* Content */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        <div className="lg:col-span-3 rounded-xl border border-neutral-200 dark:border-neutral-800 p-4">
          <Skeleton className="h-5 w-28 mb-4" />
          {Array.from({ length: 5 }).map((_, i) => (
            <JobRowSkeleton key={i} />
          ))}
        </div>
        <div className="lg:col-span-2 space-y-6">
          <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 p-4">
            <Skeleton className="h-5 w-24 mb-4" />
            <SkeletonText lines={4} />
          </div>
        </div>
      </div>
    </div>
  );
}
