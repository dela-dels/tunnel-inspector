import React from 'react';
import { useInspectorStore } from '../stores/useInspectorStore';
import { Radio } from 'lucide-react';

export const Header: React.FC = () => {
  const wsConnected = useInspectorStore((s) => s.wsConnected);
  const tunnel = useInspectorStore((s) => s.tunnel);

  return (
    <header className="h-14 border-b border-slate-800 bg-slate-900/80 backdrop-blur px-4 flex items-center justify-between sticky top-0 z-20">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-sky-500 to-indigo-600 flex items-center justify-center text-white shadow-md shadow-sky-500/10">
          <Radio className="w-4 h-4 text-white animate-pulse" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <h1 className="font-semibold text-sm tracking-tight text-white">Tunnel Inspector</h1>
            <span className="text-[10px] uppercase font-bold tracking-wider px-1.5 py-0.5 rounded bg-sky-950 text-sky-400 border border-sky-800/50">
              Cloudflared
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-3 text-xs">
        {/* Upstream target info */}
        <div className="hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-md bg-slate-800/60 border border-slate-700/50 text-slate-300">
          <span className="text-slate-400">Upstream:</span>
          <span className="font-mono text-sky-400 font-medium">{tunnel.target}</span>
        </div>

        {/* Live WebSocket connection indicator */}
        <div
          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium transition-colors ${
            wsConnected
              ? 'bg-emerald-950/80 text-emerald-300 border border-emerald-800/50'
              : 'bg-rose-950/80 text-rose-300 border border-rose-800/50'
          }`}
        >
          <span
            className={`w-2 h-2 rounded-full ${
              wsConnected ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50 animate-ping-slow' : 'bg-rose-500'
            }`}
          />
          {wsConnected ? 'Live' : 'Reconnecting...'}
        </div>
      </div>
    </header>
  );
};
