import React, { useState } from 'react';
import { useInspectorStore } from '../stores/useInspectorStore';
import { Copy, Check, ExternalLink, Globe, Server, Laptop } from 'lucide-react';

export const TunnelBanner: React.FC = () => {
  const tunnel = useInspectorStore((s) => s.tunnel);
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    if (!tunnel.url) return;
    navigator.clipboard.writeText(tunnel.url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const isConnected = tunnel.status === 'connected';

  return (
    <div className="bg-slate-900 border-b border-slate-800 p-4">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center justify-between gap-4">
        {/* Public Tunnel Section */}
        <div className="flex-1 flex flex-col sm:flex-row sm:items-center gap-3">
          <div className="flex items-center gap-2">
            <div className={`p-2 rounded-lg ${isConnected ? 'bg-sky-500/10 text-sky-400' : 'bg-amber-500/10 text-amber-400'}`}>
              <Globe className="w-5 h-5" />
            </div>
            <div>
              <div className="text-[11px] uppercase tracking-wider text-slate-400 font-semibold flex items-center gap-2">
                <span>Public Tunnel URL</span>
                <span
                  className={`inline-flex items-center px-1.5 py-0.2 rounded text-[10px] font-bold ${
                    isConnected
                      ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                      : tunnel.status === 'starting'
                      ? 'bg-amber-500/15 text-amber-400 border border-amber-500/30 animate-pulse'
                      : 'bg-slate-800 text-slate-400 border border-slate-700'
                  }`}
                >
                  {tunnel.status}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                {tunnel.url ? (
                  <>
                    <span className="font-mono text-sm md:text-base font-semibold text-sky-300 select-all">
                      {tunnel.url}
                    </span>
                    <button
                      onClick={handleCopy}
                      className="inline-flex items-center gap-1 px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 active:bg-slate-600 text-slate-300 hover:text-white text-xs transition border border-slate-700/60"
                      title="Copy public URL"
                    >
                      {copied ? (
                        <>
                          <Check className="w-3.5 h-3.5 text-emerald-400" />
                          <span className="text-emerald-400">Copied</span>
                        </>
                      ) : (
                        <>
                          <Copy className="w-3.5 h-3.5" />
                          <span>Copy</span>
                        </>
                      )}
                    </button>
                    <a
                      href={tunnel.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="p-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white transition border border-slate-700/60"
                      title="Open in browser"
                    >
                      <ExternalLink className="w-3.5 h-3.5" />
                    </a>
                  </>
                ) : (
                  <span className="text-sm text-slate-500 italic">
                    {tunnel.status === 'starting' ? 'Establishing Cloudflare tunnel...' : 'No public tunnel connected'}
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Local Targets Info */}
        <div className="flex items-center gap-4 text-xs text-slate-400 border-t md:border-t-0 md:border-l border-slate-800 pt-3 md:pt-0 md:pl-6">
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4 text-slate-500" />
            <div>
              <div className="text-[10px] uppercase tracking-wider text-slate-400">Application</div>
              <div className="font-mono text-slate-200">{tunnel.target}</div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Laptop className="w-4 h-4 text-slate-500" />
            <div>
              <div className="text-[10px] uppercase tracking-wider text-slate-400">Dashboard</div>
              <div className="font-mono text-slate-200">http://localhost:4040</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
