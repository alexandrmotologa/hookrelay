import React, { useState, useEffect } from 'react';
import { MockReceivedItem } from '../types';
import { Radio, Plus, CheckCircle2, Copy, Check, Trash2, RotateCw } from 'lucide-react';

interface MockReceiverPanelProps {
  onEndpointCreated: () => void;
}

export const MockReceiverPanel: React.FC<MockReceiverPanelProps> = ({ onEndpointCreated }) => {
  const [mode, setMode] = useState<'success' | 'error' | 'rate_limit' | 'timeout' | 'chaos'>('success');
  const [history, setHistory] = useState<MockReceivedItem[]>([]);
  const [creatingEndpoint, setCreatingEndpoint] = useState(false);
  const [copied, setCopied] = useState(false);

  const mockUrl = `${window.location.origin}/api/v1/mock-receiver?mode=${mode}`;

  const fetchHistory = async () => {
    try {
      const res = await fetch('/api/v1/mock-receiver/history');
      if (res.ok) {
        setHistory(await res.json());
      }
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    fetchHistory();
    const interval = setInterval(fetchHistory, 2000);
    return () => clearInterval(interval);
  }, []);

  const handleClearHistory = async () => {
    await fetch('/api/v1/mock-receiver/clear', { method: 'POST' });
    setHistory([]);
  };

  const handleQuickCreateEndpoint = async () => {
    setCreatingEndpoint(true);
    try {
      const res = await fetch('/api/v1/endpoints', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: `Built-in Mock (${mode.toUpperCase()})`,
          url: mockUrl,
          event_types: ['*'],
          rate_limit: 100,
        }),
      });

      if (res.ok) {
        alert(`✓ Created destination endpoint pointing to Built-in Mock Receiver in ${mode.toUpperCase()} mode!`);
        onEndpointCreated();
      }
    } catch (err: any) {
      alert(`Error creating mock endpoint: ${err.message}`);
    } finally {
      setCreatingEndpoint(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Banner */}
      <div className="bg-neutral-900 border border-neutral-800 p-5 rounded-2xl flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center space-x-2">
            <Radio className="w-5 h-5 text-emerald-400" />
            <h2 className="text-base font-bold text-white">Built-in Mock Receiver & Chaos Simulator</h2>
          </div>
          <p className="text-xs text-neutral-400">
            A built-in webhook receiver inside HookRelay. Test retries, circuit breaking, and rate limiting locally without ngrok or external services.
          </p>
        </div>

        <button
          onClick={handleQuickCreateEndpoint}
          disabled={creatingEndpoint}
          className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-4 py-2 rounded-lg transition-colors flex items-center space-x-1.5 shadow-sm whitespace-nowrap disabled:opacity-50"
        >
          <Plus className="w-4 h-4" />
          <span>Quick Create Endpoint ({mode})</span>
        </button>
      </div>

      {/* Simulator Mode Config Card */}
      <div className="bg-neutral-900/60 border border-neutral-800 p-5 rounded-2xl space-y-4 text-xs">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <span className="font-semibold text-white">Select Simulated Receiver Behavior:</span>

          <div className="flex items-center space-x-1.5 bg-neutral-950 p-1 rounded-lg border border-neutral-800">
            {(['success', 'error', 'rate_limit', 'timeout', 'chaos'] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={`px-3 py-1.5 rounded-md font-semibold transition-colors capitalize ${
                  mode === m
                    ? m === 'error' ? 'bg-rose-600 text-white'
                    : m === 'rate_limit' ? 'bg-amber-600 text-white'
                    : m === 'timeout' ? 'bg-purple-600 text-white'
                    : m === 'chaos' ? 'bg-cyan-600 text-white'
                    : 'bg-emerald-600 text-white'
                    : 'text-neutral-400 hover:text-white'
                }`}
              >
                {m === 'rate_limit' ? '429 Rate Limit' : m === 'timeout' ? '504 Timeout' : m}
              </button>
            ))}
          </div>
        </div>

        {/* URL Box */}
        <div className="flex items-center space-x-2">
          <input
            type="text"
            readOnly
            value={mockUrl}
            className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-xs font-mono text-emerald-400 select-all"
          />
          <button
            onClick={() => {
              navigator.clipboard.writeText(mockUrl);
              setCopied(true);
              setTimeout(() => setCopied(false), 2000);
            }}
            className="bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs px-3 py-2 rounded-lg transition-colors flex items-center space-x-1"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>

        {/* Mode explanation */}
        <div className="bg-neutral-950/80 border border-neutral-800/80 rounded-xl p-3 text-[11px] text-neutral-400 flex items-center space-x-2">
          <span className="font-semibold text-neutral-200 uppercase font-mono">Behavior:</span>
          <span>
            {mode === 'success' && 'Responds immediately with HTTP 200 OK and confirms received signature.'}
            {mode === 'error' && 'Simulates customer outage with HTTP 500 Internal Server Error (triggers exponential retries).'}
            {mode === 'rate_limit' && 'Responds with HTTP 429 Too Many Requests and sends Retry-After: 10s header.'}
            {mode === 'timeout' && 'Delays response by 6 seconds to trigger HookRelay’s strict 5-second connect timeout.'}
            {mode === 'chaos' && 'Randomly injects faults: 50% 200 OK, 25% 500 Error, 15% 429 Rate Limit, 10% Timeout.'}
          </span>
        </div>
      </div>

      {/* Received Webhooks Stream */}
      <div className="border border-neutral-800 bg-neutral-900/50 rounded-2xl overflow-hidden shadow-sm">
        <div className="p-4 border-b border-neutral-800 flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <h3 className="font-semibold text-sm text-white">Live Mock Receipts</h3>
            <span className="text-xs font-mono text-neutral-400">({history.length} received)</span>
          </div>

          <div className="flex items-center space-x-2">
            <button
              onClick={fetchHistory}
              className="p-1.5 text-neutral-400 hover:text-white rounded-lg"
              title="Refresh"
            >
              <RotateCw className="w-3.5 h-3.5" />
            </button>
            {history.length > 0 && (
              <button
                onClick={handleClearHistory}
                className="text-neutral-400 hover:text-rose-400 text-xs px-2 py-1 rounded hover:bg-neutral-800 flex items-center space-x-1"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span>Clear</span>
              </button>
            )}
          </div>
        </div>

        <div className="overflow-x-auto text-xs">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-neutral-800 bg-neutral-900/80 text-[11px] font-semibold uppercase tracking-wider text-neutral-400 font-mono">
                <th className="py-2.5 px-4">Status Code</th>
                <th className="py-2.5 px-4">Webhook ID</th>
                <th className="py-2.5 px-4">Signature</th>
                <th className="py-2.5 px-4">Body Preview</th>
                <th className="py-2.5 px-4">Received At</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-800/60 font-mono">
              {history.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-12 text-center text-neutral-500 font-sans">
                    No webhooks received by the mock listener yet. Send a test webhook to see it captured live!
                  </td>
                </tr>
              ) : (
                history.map((it, idx) => (
                  <tr key={idx} className="hover:bg-neutral-800/30 transition-colors">
                    <td className="py-2.5 px-4 whitespace-nowrap">
                      <span className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                        it.status_code === 200 ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' :
                        it.status_code === 429 ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' :
                        'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                      }`}>
                        {it.status_code} ({it.simulated_mode})
                      </span>
                    </td>
                    <td className="py-2.5 px-4 text-neutral-300 font-semibold truncate max-w-xs">
                      {it.webhook_id || 'none'}
                    </td>
                    <td className="py-2.5 px-4 whitespace-nowrap">
                      {it.signature_valid ? (
                        <span className="text-emerald-400 flex items-center space-x-1 font-sans font-semibold text-[11px]">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                          <span>HMAC Valid</span>
                        </span>
                      ) : (
                        <span className="text-neutral-500 font-sans text-[11px]">
                          {it.webhook_signature ? 'Present' : 'Missing'}
                        </span>
                      )}
                    </td>
                    <td className="py-2.5 px-4 text-neutral-400 truncate max-w-md">
                      {it.body || '<empty>'}
                    </td>
                    <td className="py-2.5 px-4 text-neutral-500 text-[11px] whitespace-nowrap">
                      {new Date(it.received_at).toLocaleTimeString()}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
