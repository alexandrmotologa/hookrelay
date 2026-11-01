import React, { useState } from 'react';
import { Endpoint, EndpointStatus } from '../types';
import { Plus, Shield, Globe, Key, RotateCw, Trash2, Check, Copy, ExternalLink, Zap, AlertTriangle, Eye, EyeOff, Sliders } from 'lucide-react';

interface EndpointManagerProps {
  endpoints: Endpoint[];
  onRefresh: () => void;
}

export const EndpointManager: React.FC<EndpointManagerProps> = ({ endpoints, onRefresh }) => {
  const [showAddModal, setShowAddModal] = useState(false);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [eventTypesStr, setEventTypesStr] = useState('payment.*, invoice.paid');
  const [secret, setSecret] = useState('');
  const [customHeadersStr, setCustomHeadersStr] = useState('{\n  "Authorization": "Bearer secret_token_xyz"\n}');
  const [rateLimit, setRateLimit] = useState(100);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [revealedSecrets, setRevealedSecrets] = useState<Record<string, boolean>>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [portalLink, setPortalLink] = useState<{ id: string; url: string } | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const toggleReveal = (id: string) => {
    setRevealedSecrets((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const copyText = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleCreateEndpoint = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    const eventTypes = eventTypesStr
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);

    let customHeaders: Record<string, string> = {};
    if (customHeadersStr.trim()) {
      try {
        customHeaders = JSON.parse(customHeadersStr);
      } catch (err: any) {
        setError(`Invalid Custom Headers JSON: ${err.message}`);
        setSubmitting(false);
        return;
      }
    }

    try {
      const res = await fetch('/api/v1/endpoints', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name,
          url,
          secret: secret || undefined,
          event_types: eventTypes,
          custom_headers: customHeaders,
          rate_limit: Number(rateLimit),
        }),
      });

      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Failed to create endpoint');
      }

      setShowAddModal(false);
      setName('');
      setUrl('');
      setSecret('');
      onRefresh();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this endpoint and its configuration?')) return;
    try {
      await fetch(`/api/v1/endpoints/${id}`, { method: 'DELETE' });
      onRefresh();
    } catch {
      // ignore
    }
  };

  const handleTestPing = async (id: string) => {
    setActionLoading(`ping-${id}`);
    try {
      const res = await fetch(`/api/v1/endpoints/${id}/test-ping`, { method: 'POST' });
      const data = await res.json();
      if (data.status === 'SUCCESS') {
        alert(`✓ Test ping succeeded! Returned HTTP ${data.http_status} in ${data.duration_ms}ms.`);
      } else {
        alert(`✗ Test ping failed: ${data.error_message || 'Connection failed'}`);
      }
      onRefresh();
    } catch (err: any) {
      alert(`Error triggering test ping: ${err.message}`);
    } finally {
      setActionLoading(null);
    }
  };

  const handleResetCircuitBreaker = async (id: string) => {
    setActionLoading(`reset-${id}`);
    try {
      await fetch(`/api/v1/endpoints/${id}/reset-circuit-breaker`, { method: 'POST' });
      onRefresh();
    } catch {
      // ignore
    } finally {
      setActionLoading(null);
    }
  };

  const handleRotateSecret = async (id: string) => {
    if (!confirm('Rotate secret key? A 24-hour dual-secret grace period will be activated automatically to prevent delivery loss.')) return;
    setActionLoading(`rotate-${id}`);
    try {
      await fetch(`/api/v1/endpoints/${id}/rotate-secret`, { method: 'POST' });
      onRefresh();
    } catch {
      // ignore
    } finally {
      setActionLoading(null);
    }
  };

  const handleGeneratePortalLink = async (id: string) => {
    setActionLoading(`portal-${id}`);
    try {
      const res = await fetch('/api/v1/portal/token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ endpoint_id: id, expires_in_hours: 72 }),
      });
      const data = await res.json();
      const origin = window.location.origin;
      const fullUrl = `${origin}/?portal_token=${data.token}`;
      setPortalLink({ id, url: fullUrl });
    } catch (err: any) {
      alert(`Error generating portal link: ${err.message}`);
    } finally {
      setActionLoading(null);
    }
  };

  const getStatusBadge = (status: EndpointStatus, failures: number) => {
    if (status === 'SUSPENDED') {
      return (
        <span className="inline-flex items-center space-x-1.5 px-2.5 py-0.5 rounded-full text-xs font-bold bg-rose-500/10 text-rose-400 border border-rose-500/20">
          <AlertTriangle className="w-3.5 h-3.5" />
          <span>Circuit Tripped ({failures} failures)</span>
        </span>
      );
    }
    if (status === 'ACTIVE') {
      return (
        <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
          <span>Active</span>
        </span>
      );
    }
    return (
      <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-neutral-800 text-neutral-400 border border-neutral-700">
        <span>Disabled</span>
      </span>
    );
  };

  return (
    <div className="space-y-6">
      {/* Top Banner */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-neutral-900 border border-neutral-800 p-4 rounded-xl">
        <div>
          <h2 className="text-base font-bold text-white flex items-center space-x-2">
            <Shield className="w-4 h-4 text-emerald-400" />
            <span>Customer Destination Endpoints</span>
          </h2>
          <p className="text-xs text-neutral-400 mt-0.5">
            Registered webhook listeners. Each endpoint features its own HMAC-SHA256 secret, dual-secret grace rotation, custom headers, and circuit breaker.
          </p>
        </div>

        <button
          onClick={() => setShowAddModal(true)}
          className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-3.5 py-2 rounded-lg transition-colors flex items-center space-x-1.5 shadow-sm whitespace-nowrap"
        >
          <Plus className="w-4 h-4" />
          <span>New Endpoint</span>
        </button>
      </div>

      {/* Endpoints Cards */}
      <div className="grid grid-cols-1 gap-4">
        {endpoints.length === 0 ? (
          <div className="border border-neutral-800 bg-neutral-900/40 rounded-xl p-12 text-center">
            <div className="w-12 h-12 rounded-full bg-neutral-800 flex items-center justify-center mx-auto text-neutral-400 mb-3">
              <Globe className="w-6 h-6" />
            </div>
            <h4 className="text-sm font-semibold text-neutral-200">No Destination Endpoints</h4>
            <p className="text-xs text-neutral-500 max-w-sm mx-auto mt-1 mb-4">
              Add your first customer webhook URL to start delivering cryptographically signed events.
            </p>
            <button
              onClick={() => setShowAddModal(true)}
              className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-4 py-2 rounded-lg transition-colors"
            >
              Add Endpoint
            </button>
          </div>
        ) : (
          endpoints.map((ep) => (
            <div
              key={ep.id}
              className={`border rounded-xl p-5 transition-all bg-neutral-900/60 ${
                ep.status === 'SUSPENDED'
                  ? 'border-rose-500/40 bg-rose-950/10'
                  : 'border-neutral-800 hover:border-neutral-700'
              }`}
            >
              <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-3">
                <div className="space-y-1 max-w-xl">
                  <div className="flex items-center space-x-2">
                    <h3 className="font-semibold text-white text-sm">{ep.name}</h3>
                    {getStatusBadge(ep.status, ep.consecutive_failures)}
                    <span className="text-[11px] font-mono text-neutral-500">#{ep.id}</span>
                  </div>

                  <div className="flex items-center space-x-2 text-xs text-neutral-300 font-mono">
                    <Globe className="w-3.5 h-3.5 text-neutral-500 flex-shrink-0" />
                    <span className="truncate max-w-md">{ep.url}</span>
                  </div>
                </div>

                {/* Actions */}
                <div className="flex items-center space-x-2 flex-wrap">
                  {ep.status === 'SUSPENDED' && (
                    <button
                      onClick={() => handleResetCircuitBreaker(ep.id)}
                      disabled={actionLoading === `reset-${ep.id}`}
                      className="bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold px-2.5 py-1.5 rounded-lg transition-colors flex items-center space-x-1"
                    >
                      <RotateCw className={`w-3.5 h-3.5 ${actionLoading === `reset-${ep.id}` ? 'animate-spin' : ''}`} />
                      <span>Reset Breaker</span>
                    </button>
                  )}

                  <button
                    onClick={() => handleTestPing(ep.id)}
                    disabled={actionLoading === `ping-${ep.id}`}
                    className="bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium px-2.5 py-1.5 rounded-lg transition-colors flex items-center space-x-1"
                    title="Send immediate ping webhook"
                  >
                    <Zap className="w-3.5 h-3.5 text-amber-400" />
                    <span>Test Ping</span>
                  </button>

                  <button
                    onClick={() => handleGeneratePortalLink(ep.id)}
                    disabled={actionLoading === `portal-${ep.id}`}
                    className="bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium px-2.5 py-1.5 rounded-lg transition-colors flex items-center space-x-1"
                    title="Get customer portal URL for this client"
                  >
                    <ExternalLink className="w-3.5 h-3.5 text-emerald-400" />
                    <span>Customer Portal</span>
                  </button>

                  <button
                    onClick={() => handleDelete(ep.id)}
                    className="p-1.5 text-neutral-500 hover:text-rose-400 hover:bg-neutral-800 rounded-lg transition-colors"
                    title="Delete endpoint"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Subscribed Events, Custom Headers & Secret Key Strip */}
              <div className="mt-4 pt-3 border-t border-neutral-800/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
                {/* Event & Custom Header Tags */}
                <div className="flex items-center space-x-2 flex-wrap gap-y-1">
                  <span className="text-neutral-500 text-[11px] font-semibold uppercase">Events:</span>
                  {ep.event_types && ep.event_types.length > 0 ? (
                    ep.event_types.map((ev) => (
                      <span
                        key={ev}
                        className="bg-neutral-800 text-neutral-300 font-mono text-[11px] px-2 py-0.5 rounded border border-neutral-700"
                      >
                        {ev}
                      </span>
                    ))
                  ) : (
                    <span className="text-neutral-500 font-mono text-[11px]">all (*)</span>
                  )}

                  {ep.custom_headers && Object.keys(ep.custom_headers).length > 0 && (
                    <div className="flex items-center space-x-1 ml-2">
                      <Sliders className="w-3 h-3 text-neutral-500" />
                      <span className="text-[11px] text-neutral-400 font-mono">
                        {Object.keys(ep.custom_headers).length} Custom Header(s)
                      </span>
                    </div>
                  )}

                  {/* Dual Secret Grace Period Badge */}
                  {ep.previous_secret && ep.secret_rotated_at && (
                    <span className="bg-cyan-500/10 text-cyan-400 font-mono text-[10px] px-2 py-0.5 rounded border border-cyan-500/20">
                      Dual-Sign Active (24h Grace)
                    </span>
                  )}
                </div>

                {/* Secret Key Box */}
                <div className="flex items-center space-x-2">
                  <Key className="w-3.5 h-3.5 text-neutral-500" />
                  <span className="font-mono text-neutral-300 bg-neutral-950 px-2 py-1 rounded border border-neutral-800">
                    {revealedSecrets[ep.id] ? ep.secret : `${ep.secret.slice(0, 10)}••••••••••••`}
                  </span>
                  <button
                    onClick={() => toggleReveal(ep.id)}
                    className="p-1 text-neutral-500 hover:text-neutral-300"
                    title={revealedSecrets[ep.id] ? 'Hide Secret' : 'Reveal Secret'}
                  >
                    {revealedSecrets[ep.id] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                  <button
                    onClick={() => copyText(ep.secret, `sec-${ep.id}`)}
                    className="p-1 text-neutral-500 hover:text-neutral-300"
                    title="Copy Secret"
                  >
                    {copiedId === `sec-${ep.id}` ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                  <button
                    onClick={() => handleRotateSecret(ep.id)}
                    className="text-[11px] text-neutral-400 hover:text-amber-400 underline ml-1"
                    title="Rotate secret key (with 24h dual signing grace)"
                  >
                    Rotate
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Customer Portal Link Modal */}
      {portalLink && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
          <div className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-lg p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-semibold text-white flex items-center space-x-2">
              <ExternalLink className="w-4 h-4 text-emerald-400" />
              <span>Customer Self-Serve Portal Link</span>
            </h3>
            <p className="text-xs text-neutral-400">
              Share this secure, time-limited portal link with your customer. They can inspect their delivery logs, test their receiver, and trigger manual retries without contacting your team.
            </p>

            <div className="flex items-center space-x-2">
              <input
                type="text"
                readOnly
                value={portalLink.url}
                className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-xs font-mono text-neutral-200 select-all"
              />
              <button
                onClick={() => copyText(portalLink.url, 'portal-link')}
                className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-3 py-2 rounded-lg transition-colors flex items-center space-x-1"
              >
                {copiedId === 'portal-link' ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                <span>{copiedId === 'portal-link' ? 'Copied' : 'Copy'}</span>
              </button>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setPortalLink(null)}
                className="px-4 py-1.5 text-xs font-medium text-neutral-400 hover:text-white"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Endpoint Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fadeIn">
          <div className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-lg p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-semibold text-white">Add Destination Endpoint</h3>

            {error && (
              <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs">
                {error}
              </div>
            )}

            <form onSubmit={handleCreateEndpoint} className="space-y-3 text-xs">
              <div>
                <label className="block text-neutral-300 font-medium mb-1">Friendly Name</label>
                <input
                  type="text"
                  placeholder="e.g. Acme Billing Receiver"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 focus:outline-none focus:border-neutral-700"
                />
              </div>

              <div>
                <label className="block text-neutral-300 font-medium mb-1">Destination Webhook URL *</label>
                <input
                  type="url"
                  required
                  placeholder="https://api.acme.com/webhooks"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
                />
              </div>

              <div>
                <label className="block text-neutral-300 font-medium mb-1">
                  Subscribed Event Types (comma separated)
                </label>
                <input
                  type="text"
                  placeholder="payment.*, invoice.paid, user.created"
                  value={eventTypesStr}
                  onChange={(e) => setEventTypesStr(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
                />
              </div>

              <div>
                <label className="block text-neutral-300 font-medium mb-1">
                  Custom HTTP Headers (JSON map, optional)
                </label>
                <textarea
                  rows={3}
                  placeholder='{"Authorization": "Bearer token", "X-Custom-Header": "value"}'
                  value={customHeadersStr}
                  onChange={(e) => setCustomHeadersStr(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-lg p-2.5 font-mono text-neutral-200 text-xs focus:outline-none focus:border-neutral-700"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-neutral-300 font-medium mb-1">Rate Limit (req/sec)</label>
                  <input
                    type="number"
                    min="1"
                    max="1000"
                    value={rateLimit}
                    onChange={(e) => setRateLimit(Number(e.target.value))}
                    className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 focus:outline-none focus:border-neutral-700"
                  />
                </div>

                <div>
                  <label className="block text-neutral-300 font-medium mb-1">Custom Secret (optional)</label>
                  <input
                    type="text"
                    placeholder="whsec_... (or auto)"
                    value={secret}
                    onChange={(e) => setSecret(e.target.value)}
                    className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="px-3.5 py-2 text-neutral-400 hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="bg-emerald-600 hover:bg-emerald-500 text-white font-semibold px-4 py-2 rounded-lg transition-colors shadow-sm disabled:opacity-50"
                >
                  {submitting ? 'Creating...' : 'Create Endpoint'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
