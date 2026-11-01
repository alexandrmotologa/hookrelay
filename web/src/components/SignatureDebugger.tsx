import React, { useState } from 'react';
import { ShieldCheck, CheckCircle2, AlertCircle, Copy, Check, Terminal, Clock, Key } from 'lucide-react';

export const SignatureDebugger: React.FC = () => {
  const [msgId, setMsgId] = useState('msg_demo_01h8kz99');
  const [timestamp, setTimestamp] = useState(String(Math.floor(Date.now() / 1000)));
  const [payload, setPayload] = useState('{\n  "event": "payment.succeeded",\n  "amount": 4200\n}');
  const [secret, setSecret] = useState('whsec_5pD6qF9K3b2vL8mN1xJ0wA4eR7tY8uI2oP5aS6dF7g==');
  const [signatureHeader, setSignatureHeader] = useState('');

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const copyText = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setResult(null);

    try {
      const res = await fetch('/api/v1/verifier/debug', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message_id: msgId.trim(),
          timestamp: timestamp.trim(),
          payload: payload,
          secret: secret.trim(),
          signature_header: signatureHeader.trim(),
        }),
      });

      const data = await res.json();
      setResult(data);
      if (!signatureHeader.trim() && data.expected_signature) {
        // Auto-fill provided signature for convenience
        setSignatureHeader(data.expected_signature);
      }
    } catch (err: any) {
      setResult({ error: err.message, valid: false });
    } finally {
      setLoading(false);
    }
  };

  const handleSetCurrentTimestamp = () => {
    setTimestamp(String(Math.floor(Date.now() / 1000)));
  };

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="bg-neutral-900 border border-neutral-800 p-5 rounded-2xl space-y-1">
        <div className="flex items-center space-x-2">
          <ShieldCheck className="w-5 h-5 text-emerald-400" />
          <h2 className="text-base font-bold text-white">Standard Webhooks Signature Debugger</h2>
        </div>
        <p className="text-xs text-neutral-400">
          Diagnose signature mismatches and timestamp drift. Inspect the exact string passed to HMAC-SHA256 and confirm client verifier code accuracy.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Input Form */}
        <div className="bg-neutral-900/60 border border-neutral-800 p-6 rounded-2xl space-y-4">
          <h3 className="text-sm font-semibold text-white flex items-center space-x-2">
            <Terminal className="w-4 h-4 text-neutral-400" />
            <span>Incoming Webhook Parameters</span>
          </h3>

          <form onSubmit={handleVerify} className="space-y-3.5 text-xs">
            <div>
              <label className="block text-neutral-300 font-medium mb-1">Message ID (`webhook-id`)</label>
              <input
                type="text"
                required
                value={msgId}
                onChange={(e) => setMsgId(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
              />
            </div>

            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-neutral-300 font-medium">Timestamp (`webhook-timestamp`)</label>
                <button
                  type="button"
                  onClick={handleSetCurrentTimestamp}
                  className="text-neutral-500 hover:text-emerald-400 flex items-center space-x-1"
                >
                  <Clock className="w-3 h-3" />
                  <span>Set Current Time</span>
                </button>
              </div>
              <input
                type="text"
                required
                value={timestamp}
                onChange={(e) => setTimestamp(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
              />
            </div>

            <div>
              <label className="block text-neutral-300 font-medium mb-1">Secret Key (`whsec_...` or raw bytes)</label>
              <div className="relative">
                <Key className="w-3.5 h-3.5 text-neutral-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  required
                  value={secret}
                  onChange={(e) => setSecret(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg pl-8 pr-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
                />
              </div>
            </div>

            <div>
              <label className="block text-neutral-300 font-medium mb-1">Raw JSON Body (Exact Payload)</label>
              <textarea
                rows={5}
                required
                value={payload}
                onChange={(e) => setPayload(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg p-3 text-neutral-200 font-mono text-xs focus:outline-none focus:border-neutral-700 leading-relaxed"
              />
              <p className="text-[11px] text-neutral-500 mt-0.5">Whitespace and carriage returns matter for HMAC calculation.</p>
            </div>

            <div>
              <label className="block text-neutral-300 font-medium mb-1">
                Provided Signature Header (`webhook-signature`)
              </label>
              <input
                type="text"
                placeholder="v1,g0hM9SsE+... (leave empty to compute expected)"
                value={signatureHeader}
                onChange={(e) => setSignatureHeader(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-emerald-600 hover:bg-emerald-500 text-white font-semibold py-2.5 rounded-lg transition-colors shadow-sm disabled:opacity-50"
            >
              {loading ? 'Computing...' : 'Verify & Debug Signature'}
            </button>
          </form>
        </div>

        {/* Right: Diagnostic Results */}
        <div className="bg-neutral-900/60 border border-neutral-800 p-6 rounded-2xl space-y-4">
          <h3 className="text-sm font-semibold text-white">Diagnostic Results</h3>

          {!result ? (
            <div className="border border-dashed border-neutral-800 rounded-xl p-12 text-center text-xs text-neutral-500">
              Submit the form to compute and inspect the HMAC signature.
            </div>
          ) : (
            <div className="space-y-4 text-xs">
              {/* Status Banner */}
              {result.valid ? (
                <div className="bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 p-4 rounded-xl flex items-center space-x-3">
                  <CheckCircle2 className="w-5 h-5 text-emerald-400 flex-shrink-0" />
                  <div>
                    <h4 className="font-bold text-sm text-emerald-400">Signature is 100% Valid</h4>
                    <p className="text-emerald-300/80 text-[11px] mt-0.5">The signature digest matches the payload, timestamp, and secret.</p>
                  </div>
                </div>
              ) : (
                <div className="bg-rose-500/10 border border-rose-500/20 text-rose-300 p-4 rounded-xl flex items-start space-x-3">
                  <AlertCircle className="w-5 h-5 text-rose-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <h4 className="font-bold text-sm text-rose-400">Signature Verification Failed</h4>
                    <p className="text-rose-300/80 text-[11px] mt-0.5">{result.error || 'Mismatch between calculated digest and provided signature.'}</p>
                  </div>
                </div>
              )}

              {/* Expected Digest */}
              <div className="space-y-1">
                <div className="flex items-center justify-between text-neutral-400">
                  <span>Computed Signature (`Expected`)</span>
                  <button
                    onClick={() => copyText(result.expected_signature || '', 'expected-sig')}
                    className="text-neutral-500 hover:text-white flex items-center space-x-1"
                  >
                    {copiedField === 'expected-sig' ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    <span>{copiedField === 'expected-sig' ? 'Copied' : 'Copy'}</span>
                  </button>
                </div>
                <div className="bg-neutral-950 border border-neutral-800 rounded-lg p-2.5 font-mono text-emerald-400 break-all select-all">
                  {result.expected_signature}
                </div>
              </div>

              {/* Timestamp drift indicator */}
              <div className="bg-neutral-950 border border-neutral-800 rounded-lg p-3 space-y-1">
                <div className="flex items-center justify-between">
                  <span className="text-neutral-400">Timestamp Drift:</span>
                  <span className={`font-mono font-bold ${result.timestamp_valid ? 'text-emerald-400' : 'text-rose-400'}`}>
                    {result.timestamp_drift_seconds} seconds ({result.timestamp_valid ? 'Valid ≤ 300s' : 'Expired > 300s'})
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-neutral-400">Decoded Secret Key Size:</span>
                  <span className="font-mono text-neutral-200">{result.parsed_secret_bytes_len} bytes</span>
                </div>
              </div>

              {/* PayloadToSign string */}
              <div className="space-y-1">
                <div className="flex items-center justify-between text-neutral-400">
                  <span>Exact String Passed to HMAC (`msg_id.timestamp.body`):</span>
                  <button
                    onClick={() => copyText(result.computed_payload_to_sign || '', 'payload-to-sign')}
                    className="text-neutral-500 hover:text-white flex items-center space-x-1"
                  >
                    {copiedField === 'payload-to-sign' ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    <span>{copiedField === 'payload-to-sign' ? 'Copied' : 'Copy'}</span>
                  </button>
                </div>
                <pre className="bg-neutral-950 border border-neutral-800 rounded-lg p-3 font-mono text-neutral-300 text-[11px] overflow-x-auto max-h-40 leading-relaxed whitespace-pre-wrap break-all">
                  {result.computed_payload_to_sign}
                </pre>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
