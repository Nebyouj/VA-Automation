'use client';

import { useState } from 'react';
import { Copy, Check } from 'lucide-react';

interface JobPayloadProps {
  payload?: Record<string, unknown>;
}

export function JobPayload({ payload }: JobPayloadProps) {
  const [copied, setCopied] = useState(false);

  if (!payload || Object.keys(payload).length === 0) {
    return (
      <div className="rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-950 px-4 py-3">
        <span className="text-sm text-neutral-400 dark:text-neutral-500 italic">No payload</span>
      </div>
    );
  }

  const json = JSON.stringify(payload, null, 2);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(json);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group rounded-lg border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-950 overflow-hidden">
      <button
        onClick={handleCopy}
        className="absolute top-2.5 right-2.5 p-1.5 rounded-md bg-white dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-300 opacity-0 group-hover:opacity-100 transition-all"
        aria-label="Copy payload"
      >
        {copied ? <Check size={13} /> : <Copy size={13} />}
      </button>
      <pre className="px-4 py-3 text-sm font-mono text-neutral-700 dark:text-neutral-300 overflow-x-auto leading-relaxed">
        {json}
      </pre>
    </div>
  );
}
