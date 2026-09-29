import React from 'react';
import { useInspectorStore } from '../stores/useInspectorStore';
import { Search, Pause, Play, Trash2, X } from 'lucide-react';

const METHODS = ['ALL', 'GET', 'POST', 'PUT', 'DELETE', 'PATCH'];
const STATUSES = ['ALL', '2xx', '3xx', '4xx', '5xx'];

export const RequestToolbar: React.FC = () => {
  const isPaused = useInspectorStore((s) => s.isPaused);
  const togglePaused = useInspectorStore((s) => s.togglePaused);
  const clearRequests = useInspectorStore((s) => s.clearRequests);
  const searchQuery = useInspectorStore((s) => s.searchQuery);
  const setSearchQuery = useInspectorStore((s) => s.setSearchQuery);
  const methodFilter = useInspectorStore((s) => s.methodFilter);
  const setMethodFilter = useInspectorStore((s) => s.setMethodFilter);
  const statusFilter = useInspectorStore((s) => s.statusFilter);
  const setStatusFilter = useInspectorStore((s) => s.setStatusFilter);
  const total = useInspectorStore((s) => s.total);

  const handleClear = () => {
    if (window.confirm('Clear all captured requests?')) {
      clearRequests();
    }
  };

  return (
    <div className="bg-slate-900/90 border-b border-slate-800 p-2.5 px-4 flex flex-wrap items-center justify-between gap-3 text-xs">
      {/* Search and filters */}
      <div className="flex flex-wrap items-center gap-3 flex-1 min-w-[280px]">
        {/* Search input */}
        <div className="relative flex-1 max-w-sm">
          <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search path or query..."
            className="w-full bg-slate-950 border border-slate-800 rounded-md pl-8 pr-7 py-1.5 text-xs text-slate-200 placeholder-slate-400 focus:outline-none focus:border-sky-500 transition"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>

        {/* Method filter pills */}
        <div className="flex items-center rounded-md bg-slate-950 p-0.5 border border-slate-800">
          {METHODS.map((m) => (
            <button
              key={m}
              onClick={() => setMethodFilter(m)}
              className={`px-2 py-1 rounded text-[11px] font-semibold transition ${
                methodFilter === m
                  ? 'bg-sky-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              {m}
            </button>
          ))}
        </div>

        {/* Status filter pills */}
        <div className="flex items-center rounded-md bg-slate-950 p-0.5 border border-slate-800">
          {STATUSES.map((s) => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={`px-2 py-1 rounded text-[11px] font-semibold transition ${
                statusFilter === s
                  ? 'bg-slate-700 text-white'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        <span className="text-slate-400 font-mono text-[11px] mr-1">
          {total} {total === 1 ? 'request' : 'requests'}
        </span>

        {/* Pause / Resume */}
        <button
          onClick={togglePaused}
          className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition border ${
            isPaused
              ? 'bg-amber-500/20 text-amber-300 border-amber-500/40 hover:bg-amber-500/30'
              : 'bg-slate-800 hover:bg-slate-700 text-slate-300 border-slate-700'
          }`}
          title={isPaused ? 'Resume live updates' : 'Pause live stream'}
        >
          {isPaused ? <Play className="w-3 h-3 fill-current" /> : <Pause className="w-3 h-3 fill-current" />}
          <span>{isPaused ? 'Resume' : 'Pause'}</span>
        </button>

        {/* Clear */}
        <button
          onClick={handleClear}
          className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-md bg-slate-800 hover:bg-rose-950 hover:text-rose-400 text-slate-400 border border-slate-700 hover:border-rose-900/60 transition"
          title="Clear all requests"
        >
          <Trash2 className="w-3.5 h-3.5" />
          <span>Clear</span>
        </button>
      </div>
    </div>
  );
};
