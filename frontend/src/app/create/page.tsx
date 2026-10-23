'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { PlusCircle, Loader2, Info } from 'lucide-react';
import { createJob } from '@/lib/api';
import { JOB_TYPES, JobType } from '@/lib/types';
import { useToast } from '@/components/ui/toast';

export default function CreateJobPage() {
  const router = useRouter();
  const { toast } = useToast();

  const [type, setType] = useState<JobType>('send_email');
  const [priority, setPriority] = useState<number>(5);
  const [submitting, setSubmitting] = useState(false);

  // Dynamic payload fields based on job type
  const [emailTo, setEmailTo] = useState('');
  const [emailSubject, setEmailSubject] = useState('');
  const [reportType, setReportType] = useState('monthly');
  const [reportUserId, setReportUserId] = useState('1');
  const [processFileName, setProcessFileName] = useState('');
  const [processOperation, setProcessOperation] = useState('compress');

  const getPayloadForType = () => {
    switch (type) {
      case 'send_email':
        return { to: emailTo, subject: emailSubject };
      case 'generate_report':
        return { report_type: reportType, user_id: parseInt(reportUserId, 10) || 1 };
      case 'process_file':
        return { file_name: processFileName, operation: processOperation };
      case 'failing_job':
        return {};
      default:
        return {};
    }
  };

  const isFormValid = () => {
    switch (type) {
      case 'send_email':
        return emailTo.trim() !== '';
      case 'generate_report':
        return reportType.trim() !== '' && !isNaN(parseInt(reportUserId, 10));
      case 'process_file':
        return processFileName.trim() !== '';
      case 'failing_job':
        return true;
      default:
        return false;
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isFormValid()) {
      toast('error', 'Please fill in all required fields');
      return;
    }

    setSubmitting(true);
    try {
      const payload = getPayloadForType();
      const job = await createJob({
        type,
        priority,
        payload,
      });
      
      toast('success', `Job #${job.id} created successfully`);
      router.push(`/jobs/${job.id}`);
    } catch (err) {
      toast('error', err instanceof Error ? err.message : 'Failed to create job');
      setSubmitting(false);
    }
  };

  const selectedTypeInfo = JOB_TYPES.find((t) => t.value === type);

  return (
    <div className="max-w-3xl space-y-8 animate-in fade-in duration-300">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
          Create Job
        </h1>
        <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
          Queue a new background task for execution
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-8">
        <div className="rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 overflow-hidden">
          
          <div className="p-6 space-y-8">
            {/* Type & Priority */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <div className="space-y-3">
                <label htmlFor="job-type" className="block text-sm font-medium text-neutral-900 dark:text-neutral-100">
                  Job Type
                </label>
                <div className="relative">
                  <select
                    id="job-type"
                    value={type}
                    onChange={(e) => setType(e.target.value as JobType)}
                    className="w-full appearance-none px-4 py-2.5 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors cursor-pointer"
                  >
                    {JOB_TYPES.map((t) => (
                      <option key={t.value} value={t.value}>
                        {t.label}
                      </option>
                    ))}
                  </select>
                  <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-neutral-500">
                    <svg className="h-4 w-4 fill-current" viewBox="0 0 20 20">
                      <path d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" />
                    </svg>
                  </div>
                </div>
                {selectedTypeInfo && (
                  <p className="text-xs text-neutral-500 dark:text-neutral-400 flex items-start gap-1.5 mt-2">
                    <Info size={14} className="shrink-0 mt-0.5" />
                    {selectedTypeInfo.description}
                  </p>
                )}
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <label htmlFor="priority" className="block text-sm font-medium text-neutral-900 dark:text-neutral-100">
                    Priority <span className="text-neutral-500 font-normal ml-1">(1-10)</span>
                  </label>
                  <span className="text-xs font-medium tabular-nums text-neutral-500">
                    {priority}
                  </span>
                </div>
                <input
                  id="priority"
                  type="range"
                  min="1"
                  max="10"
                  value={priority}
                  onChange={(e) => setPriority(parseInt(e.target.value, 10))}
                  className="w-full h-2 bg-neutral-200 dark:bg-neutral-800 rounded-lg appearance-none cursor-pointer accent-neutral-900 dark:accent-white"
                />
                <div className="flex justify-between text-[10px] text-neutral-400 font-medium px-1">
                  <span>Low (1)</span>
                  <span>Critical (10)</span>
                </div>
              </div>
            </div>

            <div className="h-px bg-neutral-100 dark:bg-neutral-800" />

            {/* Dynamic Payload Form */}
            <div className="space-y-5">
              <h3 className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
                Payload Configuration
              </h3>
              
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
                {type === 'send_email' && (
                  <>
                    <div className="space-y-2">
                      <label className="block text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        Recipient Email *
                      </label>
                      <input
                        type="email"
                        required
                        value={emailTo}
                        onChange={(e) => setEmailTo(e.target.value)}
                        placeholder="user@example.com"
                        className="w-full px-4 py-2 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors"
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="block text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        Subject
                      </label>
                      <input
                        type="text"
                        value={emailSubject}
                        onChange={(e) => setEmailSubject(e.target.value)}
                        placeholder="Welcome to our platform!"
                        className="w-full px-4 py-2 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors"
                      />
                    </div>
                  </>
                )}

                {type === 'generate_report' && (
                  <>
                    <div className="space-y-2">
                      <label className="block text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        Report Type *
                      </label>
                      <input
                        type="text"
                        required
                        value={reportType}
                        onChange={(e) => setReportType(e.target.value)}
                        placeholder="e.g. monthly_summary"
                        className="w-full px-4 py-2 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors"
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="block text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        User ID *
                      </label>
                      <input
                        type="number"
                        required
                        min="1"
                        value={reportUserId}
                        onChange={(e) => setReportUserId(e.target.value)}
                        className="w-full px-4 py-2 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors"
                      />
                    </div>
                  </>
                )}

                {type === 'process_file' && (
                  <>
                    <div className="space-y-2">
                      <label className="block text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        File Name *
                      </label>
                      <input
                        type="text"
                        required
                        value={processFileName}
                        onChange={(e) => setProcessFileName(e.target.value)}
                        placeholder="data-export.csv"
                        className="w-full px-4 py-2 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors"
                      />
                    </div>
                    <div className="space-y-2">
                      <label className="block text-xs font-medium text-neutral-500 dark:text-neutral-400">
                        Operation *
                      </label>
                      <div className="relative">
                        <select
                          value={processOperation}
                          onChange={(e) => setProcessOperation(e.target.value)}
                          className="w-full appearance-none px-4 py-2 text-sm rounded-lg border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-950 text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-300 dark:focus:ring-neutral-700 transition-colors cursor-pointer"
                        >
                          <option value="compress">Compress</option>
                          <option value="extract">Extract</option>
                          <option value="analyze">Analyze</option>
                        </select>
                        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-neutral-500">
                          <svg className="h-4 w-4 fill-current" viewBox="0 0 20 20">
                            <path d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" />
                          </svg>
                        </div>
                      </div>
                    </div>
                  </>
                )}

                {type === 'failing_job' && (
                  <div className="col-span-full">
                    <p className="text-sm text-neutral-500 dark:text-neutral-400 italic">
                      This job type requires no payload and is designed to fail for testing purposes.
                    </p>
                  </div>
                )}
              </div>
            </div>

            {/* Payload Preview */}
            <div className="rounded-lg bg-neutral-50 dark:bg-neutral-950 p-4 border border-neutral-100 dark:border-neutral-900">
              <h4 className="text-[11px] font-semibold text-neutral-400 dark:text-neutral-500 uppercase tracking-wider mb-2">
                Payload Preview
              </h4>
              <pre className="text-xs font-mono text-neutral-600 dark:text-neutral-300 overflow-x-auto">
                {JSON.stringify(getPayloadForType(), null, 2)}
              </pre>
            </div>
          </div>
          
          <div className="px-6 py-4 bg-neutral-50 dark:bg-neutral-900 border-t border-neutral-200 dark:border-neutral-800 flex justify-end">
            <button
              type="submit"
              disabled={submitting || !isFormValid()}
              className="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium bg-neutral-900 dark:bg-white text-white dark:text-neutral-900 rounded-lg hover:bg-neutral-800 dark:hover:bg-neutral-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <PlusCircle size={16} />
                  Queue Job
                </>
              )}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
