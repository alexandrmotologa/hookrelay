import React, { useState, useEffect } from 'react';
import { Endpoint, Delivery } from '../types';
import { Shield, Key, Eye, EyeOff, Copy, Check, RotateCw, Globe, CheckCircle2, Clock, AlertTriangle, Code2, Zap } from 'lucide-react';
import { MessagePayloadModal } from './MessagePayloadModal';

interface CustomerPortalProps {
  initialToken?: string;
}

export const CustomerPortal: React.FC<CustomerPortalProps> = ({ initialToken }) => {
  const [token, setToken] = useState(initialToken || '');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [endpoint, setEndpoint] = useState<Endpoint | null>(null);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [snippets, setSnippets] = useState<Record<string, string>>({});
  const [activeSnippetLang, setActiveSnippetLang] = useState<'nodejs' | 'python' | 'go'>('nodejs');

  const [showSecret, setShowSecret] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [inspectDelivery, setInspectDelivery] = useState<Delivery | null>(null);
  const [activeTab, setActiveTab] = useState<'deliveries' | 'verifier'>('deliveries');

  useEffect(() => {
    // Check URL params for portal_token
    const params = new URLSearchParams(window.location.search);
    const urlToken = params.get('portal_token');
    if (urlToken && !token) {
      setToken(urlToken);
      fetchPortalData(urlToken);
    } else if (token) {
      fetchPortalData(token);
    }
  }, []);

  const fetchPortalData = async (activeToken: string) => {
    if (!activeToken.trim()) return;
    setLoading(true);
    setError(null);

    try {
      // 1. Fetch portal view
      const res = await fetch(`/api/v1/portal/view?token=${encodeURIComponent(activeToken.trim())}`);
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || 'Failed to authenticate portal token');
      }

      setEndpoint(data.endpoint);
      setDeliveries(data.items || []);

      // 2. Fetch code snippets
      const snipRes = await fetch(`/api/v1/portal/snippets?token=${encodeURIComponent(activeToken.trim())}`);
      if (snipRes.ok) {
        const snipData = await snipRes.json();
        setSnippets(snipData);
      }
    } catch (err: any) {
      setError(err.message);
      setEndpoint(null);
    } finally {
      setLoading(false);
    }
  };

  const copyText = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const handleManualRetry = async (deliveryId: string) => {
    try {
      const res = await fetch(`/api/v1/portal/retry?token=${encodeURIComponent(token)}&delivery_id=${deliveryId}`, {
        method: 'POST',
      });
      if (res.ok) {
        alert('Retry initiated successfully. Check delivery logs in a moment.');
        fetchPortalData(token);
      }
    } catch (err: any) {
      alert(`Retry failed: ${err.message}`);
    }
  };

  const handleTestPing = async () => {
    if (!endpoint) return;
    try {
      const res = await fetch(`/api/v1/endpoints/${endpoint.id}/test-ping`, { method: 'POST' });
      const data = await res.json();
      if (data.status === 'SUCCESS') {
        alert(`✓ Test ping succeeded! Returned HTTP ${data.http_status} in ${data.duration_ms}ms.`);
      } else {
        alert(`✗ Test ping failed: ${data.error_message || 'Connection failed'}`);
      }
      fetchPortalData(token);
    } catch (err: any) {
      alert(`Test ping error: ${err.message}`);
    }
  };

  return (
    <div className="space-y-6">
      {/* Portal Auth Banner if not loaded */}
      {!endpoint && (
        <div className="max-w-xl mx-auto border border-neutral-800 bg-neutral-900 p-6 rounded-2xl space-y-4 shadow-xl text-xs">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-white">Customer Delivery Portal</h2>
              <p className="text-neutral-400">Enter your secure access token to inspect your webhooks.</p>
            </div>
          </div>

          {error && (
            <div className="bg-rose-500/10 border border-rose-500/20 text-rose-300 p-3 rounded-xl">
              {error}
            </div>
          )}

          <div className="space-y-3">
            <input
              type="text"
              placeholder="Paste your portal token here..."
              value={token}
              onChange={(e) => setToken(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-800 rounded-lg px-3 py-2 text-neutral-200 font-mono focus:outline-none focus:border-neutral-700"
            />
            <button
              onClick={() => fetchPortalData(token)}
              disabled={loading || !token.trim()}
              className="w-full bg-emerald-600 hover:bg-emerald-500 text-white font-semibold py-2 rounded-lg transition-colors shadow-sm disabled:opacity-50"
            >
              {loading ? 'Authenticating...' : 'Access Delivery Portal'}
            </button>
          </div>
        </div>
      )}

      {/* Authenticated Customer View */}
      {endpoint && (
        <div className="space-y-6 animate-fadeIn">
          {/* Header Card */}
          <div className="bg-neutral-900 border border-neutral-800 rounded-2xl p-6 shadow-sm space-y-4">
            <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
              <div className="space-y-1">
                <div className="flex items-center space-x-2">
                  <h2 className="text-lg font-bold text-white">{endpoint.name}</h2>
                  <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${
                    endpoint.status === 'ACTIVE'
                      ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                      : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                  }`}>
                    {endpoint.status}
                  </span>
                </div>
                <div className="flex items-center space-x-2 text-xs font-mono text-neutral-400">
                  <Globe className="w-3.5 h-3.5 text-neutral-500" />
                  <span>{endpoint.url}</span>
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <button
                  onClick={handleTestPing}
                  className="bg-neutral-800 hover:bg-neutral-700 text-white text-xs font-semibold px-3 py-1.5 rounded-lg transition-colors flex items-center space-x-1.5"
                >
                  <Zap className="w-3.5 h-3.5 text-amber-400" />
                  <span>Send Test Ping</span>
                </button>
                <button
                  onClick={() => fetchPortalData(token)}
                  className="bg-neutral-800 hover:bg-neutral-700 text-neutral-300 p-2 rounded-lg transition-colors"
                  title="Refresh logs"
                >
                  <RotateCw className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            {/* Secret key box */}
            <div className="pt-3 border-t border-neutral-800/80 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 text-xs">
              <div className="flex items-center space-x-2">
                <Key className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-neutral-400">Webhook Secret Key:</span>
                <span className="font-mono text-neutral-200 bg-neutral-950 px-2 py-1 rounded border border-neutral-800">
                  {showSecret ? endpoint.secret : `${endpoint.secret.slice(0, 10)}••••••••••••••••`}
                </span>
                <button
                  onClick={() => setShowSecret(!showSecret)}
                  className="p-1 text-neutral-400 hover:text-white"
                >
                  {showSecret ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                </button>
                <button
                  onClick={() => copyText(endpoint.secret, 'portal-secret')}
                  className="p-1 text-neutral-400 hover:text-white"
                >
                  {copiedField === 'portal-secret' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>

              <div className="flex items-center space-x-2">
                <button
                  onClick={() => setActiveTab('deliveries')}
                  className={`px-3 py-1.5 rounded-lg font-semibold transition-colors ${
                    activeTab === 'deliveries' ? 'bg-neutral-800 text-white' : 'text-neutral-400 hover:text-white'
                  }`}
                >
                  Delivery Logs ({deliveries.length})
                </button>
                <button
                  onClick={() => setActiveTab('verifier')}
                  className={`px-3 py-1.5 rounded-lg font-semibold transition-colors flex items-center space-x-1.5 ${
                    activeTab === 'verifier' ? 'bg-neutral-800 text-white' : 'text-neutral-400 hover:text-white'
                  }`}
                >
                  <Code2 className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Verify Signatures</span>
                </button>
              </div>
            </div>
          </div>

          {/* Tab 1: Delivery Logs */}
          {activeTab === 'deliveries' && (
            <div className="border border-neutral-800 bg-neutral-900/50 rounded-xl overflow-hidden shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="border-b border-neutral-800 bg-neutral-900/80 text-[11px] font-semibold uppercase tracking-wider text-neutral-400 font-mono">
                      <th className="py-3 px-4">Status</th>
                      <th className="py-3 px-4">Event Type</th>
                      <th className="py-3 px-4">Attempt</th>
                      <th className="py-3 px-4">Duration</th>
                      <th className="py-3 px-4">Timestamp</th>
                      <th className="py-3 px-4 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-800/60">
                    {deliveries.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="py-12 text-center text-neutral-500">
                          No webhook deliveries sent to your endpoint yet. Click "Send Test Ping" above to test your listener.
                        </td>
                      </tr>
                    ) : (
                      deliveries.map((d) => (
                        <tr
                          key={d.id}
                          className="hover:bg-neutral-800/40 transition-colors cursor-pointer"
                          onClick={() => setInspectDelivery(d)}
                        >
                          <td className="py-3 px-4 whitespace-nowrap">
                            {d.status === 'SUCCESS' ? (
                              <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-full font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                                <CheckCircle2 className="w-3.5 h-3.5" />
                                <span>200 OK</span>
                              </span>
                            ) : d.status === 'FAILED' ? (
                              <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-full font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
                                <AlertTriangle className="w-3.5 h-3.5" />
                                <span>Failed ({d.http_status || 'err'})</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-full font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                                <Clock className="w-3.5 h-3.5" />
                                <span>Retrying</span>
                              </span>
                            )}
                          </td>
                          <td className="py-3 px-4 font-mono font-medium text-emerald-400">
                            {d.event_type}
                          </td>
                          <td className="py-3 px-4 font-mono text-neutral-400">
                            {d.attempt_number} of {d.max_attempts}
                          </td>
                          <td className="py-3 px-4 font-mono text-neutral-300">
                            {d.duration_ms != null ? `${d.duration_ms} ms` : '-'}
                          </td>
                          <td className="py-3 px-4 text-neutral-400 font-mono text-[11px]">
                            {new Date(d.created_at).toLocaleString()}
                          </td>
                          <td className="py-3 px-4 text-right space-x-2" onClick={(e) => e.stopPropagation()}>
                            <button
                              onClick={() => setInspectDelivery(d)}
                              className="text-neutral-400 hover:text-white px-2 py-1 rounded hover:bg-neutral-800"
                            >
                              Inspect
                            </button>
                            <button
                              onClick={() => handleManualRetry(d.id)}
                              className="bg-neutral-800 hover:bg-neutral-700 text-emerald-400 px-2 py-1 rounded font-semibold inline-flex items-center space-x-1"
                            >
                              <RotateCw className="w-3 h-3" />
                              <span>Retry</span>
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Tab 2: Code Verification Snippets */}
          {activeTab === 'verifier' && (
            <div className="bg-neutral-900 border border-neutral-800 rounded-2xl p-6 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-white">How to Verify Standard Webhooks Signatures</h3>
                  <p className="text-xs text-neutral-400 mt-0.5">
                    Prevent forged webhooks and replay attacks by validating HMAC-SHA256 signatures with your secret key.
                  </p>
                </div>

                <div className="flex items-center space-x-1 bg-neutral-950 border border-neutral-800 rounded-lg p-1">
                  {(['nodejs', 'python', 'go'] as const).map((lang) => (
                    <button
                      key={lang}
                      onClick={() => setActiveSnippetLang(lang)}
                      className={`px-3 py-1 rounded text-xs font-semibold capitalize transition-colors ${
                        activeSnippetLang === lang
                          ? 'bg-neutral-800 text-white'
                          : 'text-neutral-400 hover:text-neutral-200'
                      }`}
                    >
                      {lang === 'nodejs' ? 'Node.js' : lang}
                    </button>
                  ))}
                </div>
              </div>

              {/* Code Snippet Box */}
              <div className="relative">
                <button
                  onClick={() => copyText(snippets[activeSnippetLang] || '', 'code-snippet')}
                  className="absolute right-3 top-3 bg-neutral-800 hover:bg-neutral-700 text-neutral-300 text-xs px-2.5 py-1 rounded-md transition-colors flex items-center space-x-1"
                >
                  {copiedField === 'code-snippet' ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copiedField === 'code-snippet' ? 'Copied' : 'Copy Code'}</span>
                </button>
                <pre className="bg-neutral-950 border border-neutral-800 rounded-xl p-4 font-mono text-neutral-200 text-xs overflow-x-auto leading-relaxed max-h-96">
                  {snippets[activeSnippetLang] || '// Loading snippet...'}
                </pre>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Inspect Modal */}
      {inspectDelivery && (
        <MessagePayloadModal
          delivery={inspectDelivery}
          onClose={() => setInspectDelivery(null)}
          onRetry={handleManualRetry}
        />
      )}
    </div>
  );
};
