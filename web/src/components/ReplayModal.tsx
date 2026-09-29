import React, { useState } from 'react';
import { HTTPTransaction } from '../types';
import { useInspectorStore } from '../stores/useInspectorStore';
import { RotateCcw, X, Play } from 'lucide-react';

interface ReplayModalProps {
  transaction: HTTPTransaction;
  onClose: () => void;
}

export const ReplayModal: React.FC<ReplayModalProps> = ({ transaction, onClose }) => {
  const replayRequest = useInspectorStore((s) => s.replayRequest);
  const [body, setBody] = useState(transaction.request.body || '');
  const [isReplaying, setIsReplaying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleReplay = async () => {
    setIsReplaying(true);
    setError(null);
    try {
      const result = await replayRequest(transaction.id, body);
      if (result) {
        onClose();
      } else {
        setError('Failed to replay request. Ensure the upstream application is running.');
      }
    } catch (err: any) {
      setError(err?.message || 'Replay failed');
    } finally {
      setIsReplaying(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-xl w-full max-w-2xl overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-800 bg-slate-950/50">
          <div className="flex items-center gap-2.5">
            <div className="p-1.5 rounded-lg bg-sky-500/10 text-sky-400">
              <RotateCcw className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-semibold text-sm text-white">Replay HTTP Request</h3>
              <p className="text-xs text-slate-400 font-mono mt-0.5">
                {transaction.request.method} {transaction.request.path}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content */}
        <div className="p-5 space-y-4 text-xs">
          {error && (
            <div className="p-3 rounded-lg bg-rose-500/15 border border-rose-500/30 text-rose-300">
              {error}
            </div>
          )}

          <div>
            <label className="block text-slate-400 font-medium mb-1.5">
              Request Payload (JSON / Text)
            </label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={10}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-xs text-slate-200 focus:outline-none focus:border-sky-500 transition resize-y"
              placeholder="Enter or modify request payload..."
            />
          </div>

          <div className="p-3 rounded-lg bg-slate-950/60 border border-slate-800/80 text-slate-400 text-[11px] leading-relaxed">
            This re-submits the request directly to the local upstream application and records the transaction.
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2.5 px-5 py-3.5 border-t border-slate-800 bg-slate-950/30">
          <button
            onClick={onClose}
            disabled={isReplaying}
            className="px-3.5 py-1.5 rounded-lg text-slate-300 hover:bg-slate-800 text-xs font-medium transition"
          >
            Cancel
          </button>
          <button
            onClick={handleReplay}
            disabled={isReplaying}
            className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white text-xs font-semibold shadow transition disabled:opacity-50"
          >
            {isReplaying ? (
              <>
                <RotateCcw className="w-3.5 h-3.5 animate-spin" />
                <span>Replaying...</span>
              </>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>Execute Replay</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
