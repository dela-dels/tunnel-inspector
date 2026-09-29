import React, { useEffect, useMemo, useRef } from 'react';
import { useInspectorStore } from '../stores/useInspectorStore';
import { Terminal, Copy, Check } from 'lucide-react';

export const RequestList: React.FC = () => {
  const requests = useInspectorStore((s) => s.requests);
  const selectedId = useInspectorStore((s) => s.selectedId);
  const selectRequest = useInspectorStore((s) => s.selectRequest);
  const searchQuery = useInspectorStore((s) => s.searchQuery);
  const methodFilter = useInspectorStore((s) => s.methodFilter);
  const statusFilter = useInspectorStore((s) => s.statusFilter);
  const tunnel = useInspectorStore((s) => s.tunnel);

  const [copiedCurl, setCopiedCurl] = React.useState(false);
  const listRef = useRef<HTMLDivElement>(null);

  // Filter requests
  const filteredRequests = useMemo(() => {
    return requests.filter((req) => {
      // Method filter
      if (methodFilter !== 'ALL' && req.method.toUpperCase() !== methodFilter.toUpperCase()) {
        return false;
      }
      // Status filter
      if (statusFilter !== 'ALL') {
        const cat = `${Math.floor(req.status / 100)}xx`;
        if (cat !== statusFilter) return false;
      }
      // Search filter
      if (searchQuery.trim() !== '') {
        const q = searchQuery.toLowerCase();
        const matchesPath = req.path.toLowerCase().includes(q);
        const matchesURL = req.url.toLowerCase().includes(q);
        const matchesStatus = req.status.toString().includes(q);
        const matchesMethod = req.method.toLowerCase().includes(q);
        if (!matchesPath && !matchesURL && !matchesStatus && !matchesMethod) {
          return false;
        }
      }
      return true;
    });
  }, [requests, methodFilter, statusFilter, searchQuery]);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (['input', 'textarea'].includes((e.target as HTMLElement).tagName.toLowerCase())) {
        return;
      }

      if (e.key === 'ArrowDown' || e.key === 'j') {
        e.preventDefault();
        const currentIndex = filteredRequests.findIndex((r) => r.id === selectedId);
        if (currentIndex < filteredRequests.length - 1) {
          selectRequest(filteredRequests[currentIndex + 1].id);
        }
      } else if (e.key === 'ArrowUp' || e.key === 'k') {
        e.preventDefault();
        const currentIndex = filteredRequests.findIndex((r) => r.id === selectedId);
        if (currentIndex > 0) {
          selectRequest(filteredRequests[currentIndex - 1].id);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [filteredRequests, selectedId, selectRequest]);

  // Method styling badge
  const getMethodBadge = (method: string) => {
    const m = method.toUpperCase();
    let color = 'bg-slate-800 text-slate-300';
    if (m === 'GET') color = 'bg-sky-500/15 text-sky-400 border border-sky-500/30';
    else if (m === 'POST') color = 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30';
    else if (m === 'PUT') color = 'bg-amber-500/15 text-amber-400 border border-amber-500/30';
    else if (m === 'PATCH') color = 'bg-orange-500/15 text-orange-400 border border-orange-500/30';
    else if (m === 'DELETE') color = 'bg-rose-500/15 text-rose-400 border border-rose-500/30';
    return (
      <span className={`inline-block px-1.5 py-0.5 rounded font-mono font-bold text-[10px] min-w-[46px] text-center ${color}`}>
        {m}
      </span>
    );
  };

  // Status code styling
  const getStatusBadge = (code: number) => {
    const cat = Math.floor(code / 100);
    let color = 'text-slate-400';
    if (cat === 2) color = 'text-emerald-400';
    else if (cat === 3) color = 'text-sky-400';
    else if (cat === 4) color = 'text-amber-400';
    else if (cat === 5) color = 'text-rose-400';

    return <span className={`font-mono font-bold ${color}`}>{code}</span>;
  };

  const sampleCurl = `curl -X POST "${tunnel.url || 'https://abc123.trycloudflare.com'}/hello" \\\n  -H "Content-Type: application/json" \\\n  -d '{"message": "Hello from Cloudflare Tunnel!"}'`;

  const copySampleCurl = () => {
    navigator.clipboard.writeText(sampleCurl);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  if (filteredRequests.length === 0) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 text-center text-slate-500">
        <div className="w-12 h-12 rounded-full bg-slate-900 border border-slate-800 flex items-center justify-center mb-3">
          <Terminal className="w-6 h-6 text-slate-600" />
        </div>
        <h3 className="font-semibold text-slate-300 text-sm mb-1">
          {requests.length === 0 ? 'Waiting for requests...' : 'No matching requests'}
        </h3>
        <p className="text-xs text-slate-400 max-w-sm mb-6">
          {requests.length === 0
            ? 'Send an HTTP request through the public tunnel URL to inspect it in real time.'
            : 'Try adjusting your search query or filters to see more results.'}
        </p>

        {requests.length === 0 && (
          <div className="w-full max-w-md bg-slate-950 border border-slate-800 rounded-lg p-3 text-left">
            <div className="flex items-center justify-between text-[11px] text-slate-400 mb-2 font-mono">
              <span>Quick Test cURL</span>
              <button
                onClick={copySampleCurl}
                className="flex items-center gap-1 text-slate-400 hover:text-sky-400 transition"
              >
                {copiedCurl ? (
                  <>
                    <Check className="w-3 h-3 text-emerald-400" />
                    <span className="text-emerald-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3 h-3" />
                    <span>Copy</span>
                  </>
                )}
              </button>
            </div>
            <pre className="font-mono text-xs text-sky-300/90 whitespace-pre-wrap break-all leading-relaxed">
              {sampleCurl}
            </pre>
          </div>
        )}
      </div>
    );
  }

  return (
    <div ref={listRef} className="flex-1 overflow-y-auto divide-y divide-slate-800/60 font-mono text-xs">
      {filteredRequests.map((req) => {
        const isSelected = req.id === selectedId;
        const timeStr = new Date(req.timestamp).toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
        });

        return (
          <div
            key={req.id}
            onClick={() => selectRequest(req.id)}
            className={`flex items-center justify-between px-3 py-2 cursor-pointer transition select-none ${
              isSelected
                ? 'bg-sky-500/10 border-l-2 border-sky-500'
                : 'hover:bg-slate-800/40 border-l-2 border-transparent'
            }`}
          >
            <div className="flex items-center gap-2.5 min-w-0 flex-1 mr-2">
              <span className="text-slate-400 text-[11px] shrink-0">{timeStr}</span>
              {getMethodBadge(req.method)}
              <span className="truncate text-slate-200 font-medium" title={req.path}>
                {req.path}
              </span>
            </div>

            <div className="flex items-center gap-3 shrink-0 text-right text-[11px]">
              {getStatusBadge(req.status)}
              <span className="text-slate-400 w-12 text-right">{req.duration}ms</span>
            </div>
          </div>
        );
      })}
    </div>
  );
};
