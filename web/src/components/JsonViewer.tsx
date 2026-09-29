import React, { useState } from 'react';
import { Copy, Check, WrapText } from 'lucide-react';

interface JsonViewerProps {
  content: string;
  isBinary?: boolean;
}

export const JsonViewer: React.FC<JsonViewerProps> = ({ content, isBinary }) => {
  const [copied, setCopied] = useState(false);
  const [wrap, setWrap] = useState(true);
  const [viewMode, setViewMode] = useState<'pretty' | 'raw'>('pretty');

  if (isBinary) {
    return (
      <div className="p-4 text-xs font-mono text-slate-400 bg-slate-950/60 rounded border border-slate-800">
        Binary payload ({content.length} characters base64 encoded)
      </div>
    );
  }

  if (!content || content.trim() === '') {
    return (
      <div className="p-4 text-xs italic text-slate-500 bg-slate-950/40 rounded border border-slate-900">
        No body content
      </div>
    );
  }

  let formatted = content;
  let isJson = false;

  try {
    const parsed = JSON.parse(content);
    isJson = true;
    formatted = JSON.stringify(parsed, null, 2);
  } catch {
    isJson = false;
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const displayText = viewMode === 'pretty' && isJson ? formatted : content;

  return (
    <div className="relative rounded-lg border border-slate-800 bg-slate-950 overflow-hidden font-mono text-xs">
      {/* Top action bar */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-slate-900/60 border-b border-slate-800 text-[11px] text-slate-400">
        <div className="flex items-center gap-2">
          {isJson && (
            <div className="flex items-center rounded bg-slate-950 p-0.5 border border-slate-800">
              <button
                onClick={() => setViewMode('pretty')}
                className={`px-2 py-0.5 rounded text-[10px] font-semibold transition ${
                  viewMode === 'pretty' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Pretty JSON
              </button>
              <button
                onClick={() => setViewMode('raw')}
                className={`px-2 py-0.5 rounded text-[10px] font-semibold transition ${
                  viewMode === 'raw' ? 'bg-sky-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Raw
              </button>
            </div>
          )}
          <span>{new TextEncoder().encode(content).length} bytes</span>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={() => setWrap(!wrap)}
            className={`p-1 rounded hover:bg-slate-800 transition ${wrap ? 'text-sky-400' : 'text-slate-400'}`}
            title="Toggle word wrap"
          >
            <WrapText className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={handleCopy}
            className="flex items-center gap-1 px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition"
            title="Copy to clipboard"
          >
            {copied ? (
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
      </div>

      {/* Code body */}
      <pre
        className={`p-3 text-slate-300 overflow-x-auto max-h-[450px] leading-relaxed ${
          wrap ? 'whitespace-pre-wrap break-all' : 'whitespace-pre'
        }`}
      >
        <code>{displayText}</code>
      </pre>
    </div>
  );
};
