import React, { useState } from 'react';
import { useInspectorStore } from '../stores/useInspectorStore';
import { JsonViewer } from './JsonViewer';
import { ReplayModal } from './ReplayModal';
import { RotateCcw, Copy, Check, Terminal } from 'lucide-react';

export const RequestDetail: React.FC = () => {
  const selectedTransaction = useInspectorStore((s) => s.selectedTransaction);
  const isLoadingDetail = useInspectorStore((s) => s.isLoadingDetail);
  const [activeTab, setActiveTab] = useState<'request' | 'response' | 'headers' | 'raw'>('request');
  const [showReplayModal, setShowReplayModal] = useState(false);
  const [copiedCurl, setCopiedCurl] = useState(false);

  if (isLoadingDetail) {
    return (
      <div className="flex-1 flex items-center justify-center p-8 text-slate-500 text-xs">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 border-2 border-sky-500 border-t-transparent rounded-full animate-spin" />
          <span>Loading transaction details...</span>
        </div>
      </div>
    );
  }

  if (!selectedTransaction) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 text-slate-500 text-xs text-center">
        <Terminal className="w-8 h-8 text-slate-700 mb-2 stroke-1" />
        <p className="font-medium text-slate-400">No request selected</p>
        <p className="text-slate-600 mt-1">Select an HTTP request from the list to view its inspection trace</p>
      </div>
    );
  }

  const { id, request, response, duration, timestamp } = selectedTransaction;

  // Build curl command for quick copy
  const handleCopyCurl = () => {
    let curl = `curl -X ${request.method} "${window.location.origin}${request.url}"`;
    for (const [key, vals] of Object.entries(request.headers)) {
      for (const val of vals) {
        curl += ` \\\n  -H "${key}: ${val}"`;
      }
    }
    if (request.body) {
      curl += ` \\\n  --data '${request.body.replace(/'/g, "'\\''")}'`;
    }
    navigator.clipboard.writeText(curl);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  // Status badge styling
  const statusCategory = Math.floor(response.status_code / 100);
  let statusBadgeClass = 'bg-slate-800 text-slate-300';
  if (statusCategory === 2) statusBadgeClass = 'bg-emerald-950/80 text-emerald-400 border border-emerald-800/60';
  else if (statusCategory === 3) statusBadgeClass = 'bg-sky-950/80 text-sky-400 border border-sky-800/60';
  else if (statusCategory === 4) statusBadgeClass = 'bg-amber-950/80 text-amber-400 border border-amber-800/60';
  else if (statusCategory === 5) statusBadgeClass = 'bg-rose-950/80 text-rose-400 border border-rose-800/60';

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-slate-900/50 overflow-hidden">
      {/* Top Banner / Summary */}
      <div className="p-4 border-b border-slate-800 bg-slate-950/60 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          {/* Method badge */}
          <span className="px-2.5 py-1 rounded font-mono font-bold text-xs bg-sky-950 text-sky-400 border border-sky-800/50">
            {request.method}
          </span>

          {/* Path and status */}
          <div>
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-semibold text-slate-100 break-all select-all">
                {request.path}
              </span>
            </div>
            <div className="flex items-center gap-3 text-[11px] text-slate-400 mt-1">
              <span className={`px-2 py-0.5 rounded font-mono font-bold text-[11px] ${statusBadgeClass}`}>
                {response.status_code}
              </span>
              <span>·</span>
              <span className="font-mono text-slate-300">{duration} ms</span>
              <span>·</span>
              <span className="text-slate-400">{new Date(timestamp).toLocaleTimeString()}</span>
              <span>·</span>
              <span className="text-slate-400 font-mono">ID: {id}</span>
            </div>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopyCurl}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white text-xs font-medium border border-slate-700 transition"
            title="Copy as cURL"
          >
            {copiedCurl ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-emerald-400">Copied cURL</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>cURL</span>
              </>
            )}
          </button>

          <button
            onClick={() => setShowReplayModal(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold shadow-sm transition"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            <span>Replay</span>
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center border-b border-slate-800 px-4 bg-slate-900 text-xs font-medium text-slate-400 gap-1">
        {(['request', 'response', 'headers', 'raw'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-3 py-2.5 border-b-2 capitalize transition ${
              activeTab === tab
                ? 'border-sky-500 text-sky-400 font-semibold'
                : 'border-transparent hover:text-slate-200'
            }`}
          >
            {tab}
            {tab === 'request' && request.size > 0 && (
              <span className="ml-1.5 text-[10px] text-slate-500">({formatSize(request.size)})</span>
            )}
            {tab === 'response' && response.size > 0 && (
              <span className="ml-1.5 text-[10px] text-slate-500">({formatSize(response.size)})</span>
            )}
          </button>
        ))}
      </div>

      {/* Tab Panels */}
      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {activeTab === 'request' && (
          <div className="space-y-5">
            {/* Query parameters */}
            {request.query && Object.keys(request.query).length > 0 && (
              <div>
                <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                  Query Parameters
                </h4>
                <div className="bg-slate-950 rounded-lg border border-slate-800 overflow-hidden font-mono text-xs">
                  <table className="w-full text-left">
                    <tbody>
                      {Object.entries(request.query).map(([key, values]) => (
                        <tr key={key} className="border-b border-slate-800/60 last:border-none">
                          <td className="px-3 py-2 text-slate-400 w-1/3 font-medium bg-slate-900/30">{key}</td>
                          <td className="px-3 py-2 text-slate-200 break-all select-all">
                            {values.join(', ')}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* Request Headers */}
            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Headers
              </h4>
              <div className="bg-slate-950 rounded-lg border border-slate-800 overflow-hidden font-mono text-xs max-h-56 overflow-y-auto">
                <table className="w-full text-left">
                  <tbody>
                    {Object.entries(request.headers).map(([key, values]) => (
                      <tr key={key} className="border-b border-slate-800/60 last:border-none hover:bg-slate-900/40">
                        <td className="px-3 py-1.5 text-slate-400 w-1/3 font-medium bg-slate-900/20">{key}</td>
                        <td className="px-3 py-1.5 text-slate-200 break-all select-all">
                          {values.join(', ')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Request Body */}
            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Body
              </h4>
              <JsonViewer content={request.body} isBinary={request.is_binary} />
            </div>
          </div>
        )}

        {activeTab === 'response' && (
          <div className="space-y-5">
            {/* Status and Headers */}
            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Headers
              </h4>
              <div className="bg-slate-950 rounded-lg border border-slate-800 overflow-hidden font-mono text-xs max-h-56 overflow-y-auto">
                <table className="w-full text-left">
                  <tbody>
                    {Object.entries(response.headers).map(([key, values]) => (
                      <tr key={key} className="border-b border-slate-800/60 last:border-none hover:bg-slate-900/40">
                        <td className="px-3 py-1.5 text-slate-400 w-1/3 font-medium bg-slate-900/20">{key}</td>
                        <td className="px-3 py-1.5 text-slate-200 break-all select-all">
                          {values.join(', ')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Response Body */}
            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Body
              </h4>
              <JsonViewer content={response.body} isBinary={response.is_binary} />
            </div>
          </div>
        )}

        {activeTab === 'headers' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Request Headers
              </h4>
              <div className="bg-slate-950 rounded-lg border border-slate-800 overflow-hidden font-mono text-xs">
                <table className="w-full text-left">
                  <tbody>
                    {Object.entries(request.headers).map(([key, values]) => (
                      <tr key={key} className="border-b border-slate-800/60 last:border-none">
                        <td className="px-3 py-1.5 text-slate-400 w-2/5 font-medium">{key}</td>
                        <td className="px-3 py-1.5 text-slate-200 break-all">{values.join(', ')}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Response Headers
              </h4>
              <div className="bg-slate-950 rounded-lg border border-slate-800 overflow-hidden font-mono text-xs">
                <table className="w-full text-left">
                  <tbody>
                    {Object.entries(response.headers).map(([key, values]) => (
                      <tr key={key} className="border-b border-slate-800/60 last:border-none">
                        <td className="px-3 py-1.5 text-slate-400 w-2/5 font-medium">{key}</td>
                        <td className="px-3 py-1.5 text-slate-200 break-all">{values.join(', ')}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'raw' && (
          <div className="space-y-4">
            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Raw Request
              </h4>
              <pre className="p-3 bg-slate-950 rounded-lg border border-slate-800 font-mono text-xs text-slate-300 whitespace-pre-wrap break-all leading-relaxed">
                {`${request.method} ${request.url} HTTP/1.1\n` +
                  Object.entries(request.headers)
                    .map(([k, v]) => `${k}: ${v.join(', ')}`)
                    .join('\n') +
                  (request.body ? `\n\n${request.body}` : '')}
              </pre>
            </div>

            <div>
              <h4 className="text-xs uppercase font-semibold tracking-wider text-slate-400 mb-2">
                Raw Response
              </h4>
              <pre className="p-3 bg-slate-950 rounded-lg border border-slate-800 font-mono text-xs text-slate-300 whitespace-pre-wrap break-all leading-relaxed">
                {`HTTP/1.1 ${response.status_code}\n` +
                  Object.entries(response.headers)
                    .map(([k, v]) => `${k}: ${v.join(', ')}`)
                    .join('\n') +
                  (response.body ? `\n\n${response.body}` : '')}
              </pre>
            </div>
          </div>
        )}
      </div>

      {/* Replay Modal */}
      {showReplayModal && (
        <ReplayModal
          transaction={selectedTransaction}
          onClose={() => setShowReplayModal(false)}
        />
      )}
    </div>
  );
};
