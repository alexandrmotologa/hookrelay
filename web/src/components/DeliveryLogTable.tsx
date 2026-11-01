import React from 'react';
import { Delivery, DeliveryStatus, Endpoint } from '../types';
import { CheckCircle2, AlertCircle, Clock, RotateCw, ExternalLink, Search, RefreshCw, Zap } from 'lucide-react';

interface DeliveryLogTableProps {
  deliveries: Delivery[];
  endpoints: Endpoint[];
  selectedStatus: string;
  setSelectedStatus: (s: string) => void;
  selectedEndpoint: string;
  setSelectedEndpoint: (e: string) => void;
  searchQuery: string;
  setSearchQuery: (q: string) => void;
  onInspect: (d: Delivery) => void;
  onRetry: (id: string) => void;
  onOpenSimulator: () => void;
}

export const DeliveryLogTable: React.FC<DeliveryLogTableProps> = ({
  deliveries,
  endpoints,
  selectedStatus,
  setSelectedStatus,
  selectedEndpoint,
  setSelectedEndpoint,
  searchQuery,
  setSearchQuery,
  onInspect,
  onRetry,
  onOpenSimulator,
}) => {
  const getStatusBadge = (status: DeliveryStatus, httpStatus?: number | null) => {
    switch (status) {
      case 'SUCCESS':
        return (
          <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <CheckCircle2 className="w-3.5 h-3.5" />
            <span>200 OK</span>
          </span>
        );
      case 'RETRY':
        return (
          <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-xs font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <Clock className="w-3.5 h-3.5" />
            <span>{httpStatus ? `HTTP ${httpStatus}` : 'Retrying'}</span>
          </span>
        );
      case 'FAILED':
        return (
          <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-xs font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
            <AlertCircle className="w-3.5 h-3.5" />
            <span>{httpStatus ? `HTTP ${httpStatus}` : 'DLQ Failed'}</span>
          </span>
        );
      case 'IN_FLIGHT':
        return (
          <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-xs font-semibold bg-purple-500/10 text-purple-400 border border-purple-500/20">
            <RefreshCw className="w-3.5 h-3.5 animate-spin" />
            <span>In Flight</span>
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-xs font-semibold bg-neutral-800 text-neutral-400 border border-neutral-700">
            <Clock className="w-3.5 h-3.5" />
            <span>Pending</span>
          </span>
        );
    }
  };

  const formatRelativeTime = (isoString: string) => {
    try {
      const date = new Date(isoString);
      const diffSec = Math.floor((Date.now() - date.getTime()) / 1000);
      if (diffSec < 5) return 'just now';
      if (diffSec < 60) return `${diffSec}s ago`;
      if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return isoString;
    }
  };

  return (
    <div className="space-y-4">
      {/* Controls Bar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-neutral-900 border border-neutral-800 p-3 rounded-xl">
        {/* Status Filter Buttons */}
        <div className="flex items-center space-x-1 overflow-x-auto pb-1 sm:pb-0">
          {(['ALL', 'SUCCESS', 'RETRY', 'FAILED'] as const).map((st) => (
            <button
              key={st}
              onClick={() => setSelectedStatus(st === 'ALL' ? '' : st)}
              className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-colors whitespace-nowrap ${
                (selectedStatus === '' && st === 'ALL') || selectedStatus === st
                  ? 'bg-neutral-800 text-white font-semibold shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              {st}
            </button>
          ))}
        </div>

        {/* Search & Endpoint Filter */}
        <div className="flex items-center space-x-2">
          {/* Endpoint Filter */}
          <select
            value={selectedEndpoint}
            onChange={(e) => setSelectedEndpoint(e.target.value)}
            className="bg-neutral-950 border border-neutral-800 text-neutral-300 text-xs rounded-lg px-2.5 py-1.5 focus:outline-none focus:border-neutral-700"
          >
            <option value="">All Endpoints</option>
            {endpoints.map((ep) => (
              <option key={ep.id} value={ep.id}>
                {ep.name}
              </option>
            ))}
          </select>

          {/* Search box */}
          <div className="relative flex-1 sm:w-64">
            <Search className="w-3.5 h-3.5 text-neutral-500 absolute left-2.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Search ID, event, url..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-800 text-neutral-200 text-xs rounded-lg pl-8 pr-3 py-1.5 focus:outline-none focus:border-neutral-700"
            />
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="border border-neutral-800 bg-neutral-900/50 rounded-xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-neutral-800 bg-neutral-900/80 text-[11px] font-semibold uppercase tracking-wider text-neutral-400 font-mono">
                <th className="py-3 px-4">Status</th>
                <th className="py-3 px-4">Event Type</th>
                <th className="py-3 px-4">Destination Endpoint</th>
                <th className="py-3 px-4">Attempt</th>
                <th className="py-3 px-4">Latency</th>
                <th className="py-3 px-4">Dispatched</th>
                <th className="py-3 px-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-800/60 text-xs">
              {deliveries.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center">
                    <div className="max-w-sm mx-auto space-y-3">
                      <div className="w-10 h-10 rounded-full bg-neutral-800 flex items-center justify-center mx-auto text-neutral-400">
                        <Zap className="w-5 h-5" />
                      </div>
                      <h4 className="text-sm font-semibold text-neutral-300">No Webhook Deliveries</h4>
                      <p className="text-xs text-neutral-500">
                        No delivery tasks match the selected filter. Publish a test event to see real-time HMAC signing and exponential retry state in action.
                      </p>
                      <button
                        onClick={onOpenSimulator}
                        className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-3.5 py-1.5 rounded-lg transition-colors inline-flex items-center space-x-1.5"
                      >
                        <span>Publish Test Webhook</span>
                      </button>
                    </div>
                  </td>
                </tr>
              ) : (
                deliveries.map((d) => (
                  <tr
                    key={d.id}
                    className="hover:bg-neutral-800/40 transition-colors group cursor-pointer"
                    onClick={() => onInspect(d)}
                  >
                    <td className="py-3 px-4 whitespace-nowrap">
                      {getStatusBadge(d.status, d.http_status)}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <span className="font-mono text-emerald-400 font-medium bg-emerald-500/5 px-2 py-0.5 rounded border border-emerald-500/10">
                        {d.event_type}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex flex-col">
                        <span className="font-medium text-neutral-200">{d.endpoint_name}</span>
                        <span className="text-[11px] font-mono text-neutral-500 truncate max-w-xs">{d.endpoint_url}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap font-mono text-neutral-400">
                      {d.attempt_number} / {d.max_attempts}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap font-mono text-neutral-300">
                      {d.duration_ms != null ? `${d.duration_ms} ms` : '-'}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap text-neutral-400 text-[11px]">
                      {formatRelativeTime(d.created_at)}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap text-right space-x-1.5" onClick={(e) => e.stopPropagation()}>
                      <button
                        onClick={() => onInspect(d)}
                        className="p-1.5 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded transition-colors inline-flex items-center"
                        title="Inspect Request & Response Payload"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => onRetry(d.id)}
                        className="p-1.5 text-neutral-400 hover:text-emerald-400 hover:bg-neutral-800 rounded transition-colors inline-flex items-center"
                        title="Trigger Immediate Manual Replay"
                      >
                        <RotateCw className="w-3.5 h-3.5" />
                      </button>
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
