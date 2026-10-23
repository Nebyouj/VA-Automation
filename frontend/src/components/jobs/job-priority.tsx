interface JobPriorityProps {
  priority: number;
}

export function JobPriority({ priority }: JobPriorityProps) {
  const getLabel = (p: number) => {
    if (p >= 8) return { text: 'Critical', className: 'text-red-600 dark:text-red-400' };
    if (p >= 5) return { text: 'High', className: 'text-amber-600 dark:text-amber-400' };
    if (p >= 3) return { text: 'Medium', className: 'text-blue-600 dark:text-blue-400' };
    return { text: 'Low', className: 'text-neutral-500 dark:text-neutral-400' };
  };

  const label = getLabel(priority);

  return (
    <span className={`text-xs font-medium tabular-nums ${label.className}`}>
      P{priority}
    </span>
  );
}
