import React, { useState } from 'react';
import { Delivery } from '../types';
import { X, Copy, Check, RotateCw, ArrowRight, ShieldCheck, Globe, Clock, Terminal } from 'lucide-react';

interface MessagePayloadModalProps {
  delivery: Delivery | null;
  onClose: () => void;
  onRetry: (id: string) => void;
}

export const MessagePayloadModal: React.FC<MessagePayloadModalProps> = ({
  delivery,
  onClose,
  onRetry,
}) => {
  const [activeTab, setActiveTab] = useState<'request' | 'response'>('request');
  const [copiedField, setCopiedField] = useState<string | null>(null);

  if (!delivery) return null;

  const copyToClipboard = (text: string, fieldName: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const formatJSON = (raw: string | undefined) => {
    if (!raw) return '';
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  };

  let parsedReqHeaders: Record<string, string> = {};
  if (delivery.request_headers) {
    try {
      const parsed = JSON.parse(delivery.request_headers);
      for (const k in parsed) {
        if (Array.isArray(parsed[k])) {
          parsedReqHeaders[k] = parsed[k].join(', ');
        } else {
          parsedReqHeaders[k] = String(parsed[k]);
        }
      }
    } catch {
      // ignore
    }
  }

  let parsedRespHeaders: Record<string, string> = {};
  if (delivery.response_headers) {
    try {
      parsedRespHeaders = JSON.parse(delivery.response_headers);
    } catch {
      // ignore
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fadeIn">
      <div className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Modal Header */}
        <div className="px-6 py-4 border-b border-neutral-800 flex items-center justify-between bg-neutral-900/80">
          <div className="flex items-center space-x-3">
            <span className="font-mono text-xs px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-bold">
              {delivery.event_type}
            </span>
            <h3 className="font-semibold text-white text-base">Delivery Details</h3>
            <span className="text-xs text-neutral-400 font-mono">#{delivery.id}</span>
          </div>

          <div className="flex items-center space-x-2">
            <button
              onClick={() => onRetry(delivery.id)}
              className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-3 py-1.5 rounded-lg transition-colors flex items-center space-x-1.5 shadow-sm"
            >
              <RotateCw className="w-3.5 h-3.5" />
              <span>Retry Now</span>
            </button>
            <button
              onClick={onClose}
              className="p-1.5 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded-lg transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Quick summary strip */}
        <div className="px-6 py-3 bg-neutral-950/70 border-b border-neutral-800 flex flex-wrap items-center justify-between gap-3 text-xs">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-1.5 text-neutral-300">
              <Globe className="w-3.5 h-3.5 text-neutral-500" />
              <span className="font-mono truncate max-w-sm">{delivery.endpoint_url}</span>
            </div>
            <div className="flex items-center space-x-1.5 text-neutral-400 font-mono">
              <Clock className="w-3.5 h-3.5 text-neutral-500" />
              <span>{delivery.duration_ms != null ? `${delivery.duration_ms} ms` : '-'}</span>
            </div>
          </div>

          <div className="flex items-center space-x-3">
            <span className="text-neutral-400">Attempt:</span>
            <span className="font-mono text-neutral-200 font-semibold">{delivery.attempt_number} of {delivery.max_attempts}</span>
            <span className={`px-2 py-0.5 rounded text-[11px] font-bold ${
              delivery.status === 'SUCCESS' ? 'bg-emerald-500/20 text-emerald-400' :
              delivery.status === 'FAILED' ? 'bg-rose-500/20 text-rose-400' :
              'bg-amber-500/20 text-amber-400'
            }`}>
              {delivery.status} {delivery.http_status ? `(${delivery.http_status})` : ''}
            </span>
          </div>
        </div>

        {/* Tabs: Request vs Response */}
        <div className="px-6 pt-3 flex items-center space-x-2 border-b border-neutral-800 bg-neutral-900">
          <button
            onClick={() => setActiveTab('request')}
            className={`pb-2.5 px-3 text-xs font-semibold flex items-center space-x-2 border-b-2 transition-colors ${
              activeTab === 'request'
                ? 'border-emerald-500 text-white'
                : 'border-transparent text-neutral-400 hover:text-neutral-200'
            }`}
          >
            <ArrowRight className="w-3.5 h-3.5" />
            <span>Outbound Request</span>
          </button>
          <button
            onClick={() => setActiveTab('response')}
            className={`pb-2.5 px-3 text-xs font-semibold flex items-center space-x-2 border-b-2 transition-colors ${
              activeTab === 'response'
                ? 'border-emerald-500 text-white'
                : 'border-transparent text-neutral-400 hover:text-neutral-200'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>Returned Response</span>
          </button>
        </div>

        {/* Tab Content Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
          {activeTab === 'request' ? (
            <>
              {/* Standard Webhooks Headers */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2 text-neutral-300 font-semibold">
                    <ShieldCheck className="w-4 h-4 text-emerald-400" />
                    <span>Cryptographic Standard Webhooks Headers</span>
                  </div>
                  <span className="text-[11px] text-neutral-500 font-mono">HMAC-SHA256</span>
                </div>
                <div className="bg-neutral-950 border border-neutral-800 rounded-xl p-3 font-mono divide-y divide-neutral-900">
                  {Object.keys(parsedReqHeaders).length > 0 ? (
                    Object.entries(parsedReqHeaders).map(([k, v]) => (
                      <div key={k} className="py-1.5 flex items-start justify-between gap-3 text-neutral-300">
                        <span className="text-emerald-400 font-semibold whitespace-nowrap">{k}:</span>
                        <span className="text-neutral-300 break-all text-right">{v}</span>
                      </div>
                    ))
                  ) : (
                    <div className="text-neutral-500 py-1">Headers not yet dispatched or recorded.</div>
                  )}
                </div>
              </div>

              {/* Request Payload */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-neutral-300 font-semibold">JSON Request Payload</span>
                  <button
                    onClick={() => copyToClipboard(formatJSON(delivery.payload), 'payload')}
                    className="p-1 px-2 rounded hover:bg-neutral-800 text-neutral-400 hover:text-white transition-colors flex items-center space-x-1"
                  >
                    {copiedField === 'payload' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                    <span>{copiedField === 'payload' ? 'Copied' : 'Copy'}</span>
                  </button>
                </div>
                <pre className="bg-neutral-950 border border-neutral-800 rounded-xl p-4 font-mono text-neutral-200 overflow-x-auto text-xs leading-relaxed max-h-60">
                  {formatJSON(delivery.payload)}
                </pre>
              </div>
            </>
          ) : (
            <>
              {/* Response Status & Error Alert */}
              {delivery.error_message && (
                <div className="bg-rose-500/10 border border-rose-500/20 text-rose-300 p-3 rounded-xl flex items-start space-x-2">
                  <span className="font-bold">Error:</span>
                  <span>{delivery.error_message}</span>
                </div>
              )}

              {/* Response Headers */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-neutral-300 font-semibold">Response Headers</span>
                </div>
                <div className="bg-neutral-950 border border-neutral-800 rounded-xl p-3 font-mono divide-y divide-neutral-900 max-h-48 overflow-y-auto">
                  {Object.keys(parsedRespHeaders).length > 0 ? (
                    Object.entries(parsedRespHeaders).map(([k, v]) => (
                      <div key={k} className="py-1 flex items-start justify-between gap-3 text-neutral-300">
                        <span className="text-neutral-400 font-semibold whitespace-nowrap">{k}:</span>
                        <span className="text-neutral-300 break-all text-right">{v}</span>
                      </div>
                    ))
                  ) : (
                    <div className="text-neutral-500 py-1">No response headers returned (e.g. connection timeout).</div>
                  )}
                </div>
              </div>

              {/* Response Body Preview */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-neutral-300 font-semibold">Response Body Preview (first 4KB)</span>
                  {delivery.response_body && (
                    <button
                      onClick={() => copyToClipboard(delivery.response_body || '', 'resp')}
                      className="p-1 px-2 rounded hover:bg-neutral-800 text-neutral-400 hover:text-white transition-colors flex items-center space-x-1"
                    >
                      {copiedField === 'resp' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      <span>{copiedField === 'resp' ? 'Copied' : 'Copy'}</span>
                    </button>
                  )}
                </div>
                <pre className="bg-neutral-950 border border-neutral-800 rounded-xl p-4 font-mono text-neutral-200 overflow-x-auto text-xs leading-relaxed max-h-56">
                  {delivery.response_body || '<Empty response body>'}
                </pre>
              </div>

              {/* Next Retry Information */}
              {delivery.next_retry_at && (
                <div className="bg-amber-500/10 border border-amber-500/20 text-amber-300 p-3 rounded-xl flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Clock className="w-4 h-4 text-amber-400" />
                    <span>Next automated retry scheduled at:</span>
                  </div>
                  <span className="font-mono font-bold text-xs">{new Date(delivery.next_retry_at).toLocaleString()}</span>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
