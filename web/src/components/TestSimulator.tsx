import React, { useState } from 'react';
import { Endpoint } from '../types';
import { Send, CheckCircle2, AlertCircle, X, Copy, Check } from 'lucide-react';

interface TestSimulatorProps {
  endpoints: Endpoint[];
  onClose: () => void;
  onSuccess: () => void;
}

const PRESET_PAYLOADS: Record<string, any> = {
  'payment.succeeded': {
    id: 'pay_99210',
    amount: 8900,
    currency: 'usd',
    customer: {
      id: 'cus_88192',
      email: 'alex@example.com',
      name: 'Alexandr M',
    },
    status: 'succeeded',
    created: Math.floor(Date.now() / 1000),
  },
  'invoice.paid': {
    id: 'in_882910',
    number: 'INV-2026-0042',
    total: 12500,
    currency: 'eur',
    paid_at: new Date().toISOString(),
    lines: [
      { description: 'HookRelay Enterprise License', quantity: 1, amount: 12500 }
    ],
  },
  'user.signup': {
    user_id: 'usr_01h89...',
    email: 'dev@company.org',
    role: 'owner',
    plan: 'production_cluster',
  },
};

export const TestSimulator: React.FC<TestSimulatorProps> = ({ endpoints, onClose, onSuccess }) => {
  const [eventType, setEventType] = useState('payment.succeeded');
  const [endpointId, setEndpointId] = useState('');
  const [payloadStr, setPayloadStr] = useState(
    JSON.stringify(PRESET_PAYLOADS['payment.succeeded'], null, 2)
  );
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<any | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleSelectPreset = (type: string) => {
    setEventType(type);
    if (PRESET_PAYLOADS[type]) {
      setPayloadStr(JSON.stringify(PRESET_PAYLOADS[type], null, 2));
    }
  };

  const handlePublish = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setResult(null);

    let parsedPayload;
    try {
      parsedPayload = JSON.parse(payloadStr);
    } catch (err: any) {
      setError(`Invalid JSON syntax: ${err.message}`);
      setSubmitting(false);
      return;
    }

    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (idempotencyKey.trim()) {
        headers['Idempotency-Key'] = idempotencyKey.trim();
      }

      const res = await fetch('/api/v1/publish', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          event_type: eventType,
          payload: parsedPayload,
          endpoint_id: endpointId || undefined,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Publish failed');
      }

      setResult(data);
      onSuccess();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fadeIn">
      <div className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-neutral-800 flex items-center justify-between bg-neutral-900/80">
          <div className="flex items-center space-x-2">
            <div className="w-7 h-7 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Send className="w-4 h-4" />
            </div>
            <h3 className="font-semibold text-white text-base">Webhook Dispatch Simulator</h3>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded-lg transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form Body */}
        <div className="p-6 overflow-y-auto space-y-4 text-xs">
          {error && (
            <div className="bg-rose-500/10 border border-rose-500/20 text-rose-300 p-3 rounded-xl flex items-center space-x-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {result && (
            <div className="bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 p-4 rounded-xl space-y-2">
              <div className="flex items-center space-x-2 font-semibold text-sm text-emerald-400">
                <CheckCircle2 className="w-4 h-4" />
                <span>Event Ingested in &lt;2ms</span>
              </div>
              <div className="font-mono text-[11px] text-neutral-300 space-y-1">
                <div>Message ID: <span className="text-white font-bold">{result.message_id}</span></div>
                <div>Deliveries Queued: <span className="text-emerald-400 font-bold">{result.deliveries_queued}</span></div>
                {result.deduplicated && <div className="text-amber-400">Deduplicated via Idempotency-Key</div>}
              </div>
            </div>
          )}

          <form onSubmit={handlePublish} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {/* Event Type */}
              <div>
                <label className="block text-neutral-300 font-medium mb-1">Event Type *</label>
                <input
                  type="text"
                  required
                  value={eventType}
                  onChange={(e) => setEventType(e.target.value)}
                  placeholder="e.g. payment.succeeded"
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
                />
              </div>

              {/* Destination Endpoint */}
              <div>
                <label className="block text-neutral-300 font-medium mb-1">Target Endpoint</label>
                <select
                  value={endpointId}
                  onChange={(e) => setEndpointId(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 focus:outline-none focus:border-neutral-700"
                >
                  <option value="">Fan-out to all matching subscribers</option>
                  {endpoints.map((ep) => (
                    <option key={ep.id} value={ep.id}>
                      {ep.name} ({ep.url})
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Presets */}
            <div className="flex items-center space-x-2">
              <span className="text-neutral-500 font-medium">Presets:</span>
              {Object.keys(PRESET_PAYLOADS).map((preset) => (
                <button
                  type="button"
                  key={preset}
                  onClick={() => handleSelectPreset(preset)}
                  className={`px-2.5 py-1 rounded text-[11px] font-mono transition-colors ${
                    eventType === preset
                      ? 'bg-neutral-800 text-emerald-400 border border-emerald-500/30'
                      : 'bg-neutral-950 text-neutral-400 hover:text-white border border-neutral-800'
                  }`}
                >
                  {preset}
                </button>
              ))}
            </div>

            {/* Idempotency Key */}
            <div>
              <label className="block text-neutral-300 font-medium mb-1">
                Idempotency Key (optional)
              </label>
              <input
                type="text"
                placeholder="evt_tx_12345"
                value={idempotencyKey}
                onChange={(e) => setIdempotencyKey(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
              />
            </div>

            {/* Payload Editor */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-neutral-300 font-medium">JSON Payload</label>
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(payloadStr);
                    setCopied(true);
                    setTimeout(() => setCopied(false), 2000);
                  }}
                  className="text-neutral-500 hover:text-white flex items-center space-x-1"
                >
                  {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copied ? 'Copied' : 'Copy'}</span>
                </button>
              </div>
              <textarea
                rows={7}
                value={payloadStr}
                onChange={(e) => setPayloadStr(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg p-3 font-mono text-neutral-200 text-xs focus:outline-none focus:border-neutral-700 leading-relaxed"
              />
            </div>

            <div className="flex items-center justify-end space-x-2 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="px-3.5 py-2 text-neutral-400 hover:text-white"
              >
                Close
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="bg-emerald-600 hover:bg-emerald-500 text-white font-semibold px-4 py-2 rounded-lg transition-colors flex items-center space-x-1.5 shadow-sm disabled:opacity-50"
              >
                <Send className="w-3.5 h-3.5" />
                <span>{submitting ? 'Publishing...' : 'Publish Event'}</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
