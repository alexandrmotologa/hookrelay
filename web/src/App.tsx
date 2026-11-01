import { useState, useEffect, useCallback } from 'react';
import { Delivery, Endpoint, StatsSummary } from './types';
import { Header, ActiveTab } from './components/Header';
import { DeliveryLogTable } from './components/DeliveryLogTable';
import { EndpointManager } from './components/EndpointManager';
import { DLQView } from './components/DLQView';
import { TestSimulator } from './components/TestSimulator';
import { CustomerPortal } from './components/CustomerPortal';
import { MessagePayloadModal } from './components/MessagePayloadModal';
import { MockReceiverPanel } from './components/MockReceiverPanel';
import { SignatureDebugger } from './components/SignatureDebugger';
import { CheckCircle2, Clock, AlertTriangle, Layers, Activity, ShieldAlert, ArrowUpRight } from 'lucide-react';

export function App() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('deliveries');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [sseConnected, setSseConnected] = useState(false);

  const [stats, setStats] = useState<StatsSummary | null>(null);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);

  const [selectedStatus, setSelectedStatus] = useState<string>('');
  const [selectedEndpoint, setSelectedEndpoint] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');

  const [inspectDelivery, setInspectDelivery] = useState<Delivery | null>(null);
  const [showSimulator, setShowSimulator] = useState(false);

  // Check URL query parameters for portal_token on mount
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get('portal_token')) {
      setActiveTab('portal');
    }
  }, []);

  const fetchData = useCallback(async () => {
    try {
      // 1. Fetch stats
      const statsRes = await fetch('/api/v1/stats');
      if (statsRes.ok) {
        const statsData = await statsRes.json();
        setStats(statsData);
      }

      // 2. Fetch endpoints
      const epRes = await fetch('/api/v1/endpoints');
      if (epRes.ok) {
        const epData = await epRes.json();
        setEndpoints(epData || []);
      }

      // 3. Fetch deliveries
      const queryParams = new URLSearchParams();
      if (selectedStatus) queryParams.set('status', selectedStatus);
      if (selectedEndpoint) queryParams.set('endpoint_id', selectedEndpoint);
      if (searchQuery.trim()) queryParams.set('search', searchQuery.trim());
      queryParams.set('limit', '50');

      const delRes = await fetch(`/api/v1/deliveries?${queryParams.toString()}`);
      if (delRes.ok) {
        const delData = await delRes.json();
        setDeliveries(delData.items || []);
      }
    } catch {
      // ignore network errors
    }
  }, [selectedStatus, selectedEndpoint, searchQuery]);

  // Connect Server-Sent Events (SSE) for instant live updates (<10ms)
  useEffect(() => {
    fetchData();

    let eventSource: EventSource | null = null;
    try {
      eventSource = new EventSource('/api/v1/events/stream');

      eventSource.addEventListener('connected', () => {
        setSseConnected(true);
      });

      eventSource.addEventListener('delivery', (e) => {
        try {
          const updatedDel: Delivery = JSON.parse(e.data);
          setDeliveries((prev) => {
            const idx = prev.findIndex((d) => d.id === updatedDel.id);
            if (idx >= 0) {
              const clone = [...prev];
              clone[idx] = updatedDel;
              return clone;
            } else {
              return [updatedDel, ...prev];
            }
          });
          // Refresh stats
          fetch('/api/v1/stats').then((r) => r.json()).then(setStats).catch(() => {});
        } catch {
          // ignore
        }
      });

      eventSource.onerror = () => {
        setSseConnected(false);
      };
    } catch {
      setSseConnected(false);
    }

    // Interval poller as resilient fallback
    const interval = setInterval(() => {
      if (autoRefresh) {
        fetchData();
      }
    }, 4000);

    return () => {
      if (eventSource) eventSource.close();
      clearInterval(interval);
    };
  }, [fetchData, autoRefresh]);

  const handleManualRetry = async (deliveryId: string) => {
    try {
      const res = await fetch(`/api/v1/deliveries/${deliveryId}/replay`, { method: 'POST' });
      if (res.ok) {
        fetchData();
        if (inspectDelivery && inspectDelivery.id === deliveryId) {
          const dRes = await fetch(`/api/v1/deliveries/${deliveryId}`);
          if (dRes.ok) {
            setInspectDelivery(await dRes.json());
          }
        }
      }
    } catch {
      // ignore
    }
  };

  const failedDeliveries = deliveries.filter((d) => d.status === 'FAILED');

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100 flex flex-col">
      {/* Top Header */}
      <Header
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        stats={stats}
        autoRefresh={autoRefresh}
        setAutoRefresh={setAutoRefresh}
        onRefresh={fetchData}
        onOpenSimulator={() => setShowSimulator(true)}
        sseConnected={sseConnected}
      />

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
        {/* KPI Summary Cards */}
        {activeTab !== 'portal' && (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
            {/* Total Ingested */}
            <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-3.5 space-y-1">
              <div className="flex items-center justify-between text-neutral-400 text-xs">
                <span>Ingested</span>
                <Layers className="w-3.5 h-3.5 text-neutral-500" />
              </div>
              <div className="text-xl font-bold font-mono text-white">
                {stats ? stats.total_messages : 0}
              </div>
            </div>

            {/* Total Dispatched */}
            <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-3.5 space-y-1">
              <div className="flex items-center justify-between text-neutral-400 text-xs">
                <span>Dispatches</span>
                <ArrowUpRight className="w-3.5 h-3.5 text-neutral-500" />
              </div>
              <div className="text-xl font-bold font-mono text-white">
                {stats ? stats.total_deliveries : 0}
              </div>
            </div>

            {/* Success Rate */}
            <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-3.5 space-y-1">
              <div className="flex items-center justify-between text-neutral-400 text-xs">
                <span>Success Rate</span>
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
              </div>
              <div className="text-xl font-bold font-mono text-emerald-400">
                {stats ? `${stats.success_rate.toFixed(1)}%` : '100%'}
              </div>
            </div>

            {/* Retrying */}
            <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-3.5 space-y-1">
              <div className="flex items-center justify-between text-neutral-400 text-xs">
                <span>Retrying</span>
                <Clock className="w-3.5 h-3.5 text-amber-400" />
              </div>
              <div className="text-xl font-bold font-mono text-amber-400">
                {stats ? stats.retrying_count : 0}
              </div>
            </div>

            {/* Dead Letter (DLQ) */}
            <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-3.5 space-y-1">
              <div className="flex items-center justify-between text-neutral-400 text-xs">
                <span>Dead Letter</span>
                <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />
              </div>
              <div className="text-xl font-bold font-mono text-rose-400">
                {stats ? stats.failed_count : 0}
              </div>
            </div>

            {/* Latency / Circuit Breaker */}
            <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-3.5 space-y-1">
              <div className="flex items-center justify-between text-neutral-400 text-xs">
                <span>Avg Latency</span>
                <Activity className="w-3.5 h-3.5 text-emerald-400" />
              </div>
              <div className="text-xl font-bold font-mono text-neutral-200">
                {stats ? `${stats.avg_duration_ms.toFixed(0)} ms` : '0 ms'}
              </div>
            </div>
          </div>
        )}

        {/* Warning Banner if Circuit Breakers are Tripped */}
        {stats && stats.suspended_endpoints > 0 && activeTab !== 'portal' && (
          <div className="bg-rose-950/20 border border-rose-500/30 rounded-xl p-4 flex items-center justify-between text-xs text-rose-300">
            <div className="flex items-center space-x-2">
              <ShieldAlert className="w-4 h-4 text-rose-400 flex-shrink-0" />
              <span>
                <strong>Circuit Breaker Alert:</strong> {stats.suspended_endpoints} destination endpoint(s) have been suspended after 50 consecutive delivery failures to protect your resources.
              </span>
            </div>
            <button
              onClick={() => setActiveTab('endpoints')}
              className="bg-rose-600 hover:bg-rose-500 text-white font-semibold px-3 py-1 rounded-lg transition-colors whitespace-nowrap ml-4"
            >
              View Endpoints
            </button>
          </div>
        )}

        {/* View Switcher */}
        {activeTab === 'deliveries' && (
          <DeliveryLogTable
            deliveries={deliveries}
            endpoints={endpoints}
            selectedStatus={selectedStatus}
            setSelectedStatus={setSelectedStatus}
            selectedEndpoint={selectedEndpoint}
            setSelectedEndpoint={setSelectedEndpoint}
            searchQuery={searchQuery}
            setSearchQuery={setSearchQuery}
            onInspect={(d) => setInspectDelivery(d)}
            onRetry={handleManualRetry}
            onOpenSimulator={() => setShowSimulator(true)}
          />
        )}

        {activeTab === 'endpoints' && (
          <EndpointManager
            endpoints={endpoints}
            onRefresh={fetchData}
          />
        )}

        {activeTab === 'dlq' && (
          <DLQView
            failedDeliveries={failedDeliveries}
            onInspect={(d) => setInspectDelivery(d)}
            onRetry={handleManualRetry}
            onRefresh={fetchData}
          />
        )}

        {activeTab === 'mock' && (
          <MockReceiverPanel
            onEndpointCreated={fetchData}
          />
        )}

        {activeTab === 'debugger' && (
          <SignatureDebugger />
        )}

        {activeTab === 'portal' && (
          <CustomerPortal />
        )}
      </main>

      {/* Simulator Modal */}
      {showSimulator && (
        <TestSimulator
          endpoints={endpoints}
          onClose={() => setShowSimulator(false)}
          onSuccess={() => {
            fetchData();
            setActiveTab('deliveries');
          }}
        />
      )}

      {/* Payload Inspection Modal */}
      {inspectDelivery && (
        <MessagePayloadModal
          delivery={inspectDelivery}
          onClose={() => setInspectDelivery(null)}
          onRetry={handleManualRetry}
        />
      )}
    </div>
  );
}

export default App;
