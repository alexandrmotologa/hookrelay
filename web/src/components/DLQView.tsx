import React, { useState } from 'react';
import { Delivery } from '../types';
import { AlertTriangle, RotateCw, CheckCircle2, ShieldAlert } from 'lucide-react';

interface DLQViewProps {
  failedDeliveries: Delivery[];
  onInspect: (d: Delivery) => void;
  onRetry: (id: string) => void;
  onRefresh: () => void;
}

export const DLQView: React.FC<DLQViewProps> = ({
  failedDeliveries,
  onInspect,
  onRetry,
  onRefresh,
}) => {
  const [replayingAll, setReplayingAll] = useState(false);

  const handleReplayAll = async () => {
    if (!confirm(`Replay all ${failedDeliveries.length} dead letter deliveries immediately?`)) return;
    setReplayingAll(true);
    try {
      const res = await fetch('/api/v1/deliveries/replay-dlq', { method: 'POST' });
      const data = await res.json();
      alert(`Queued ${data.replayed_count || 0} failed webhooks for immediate redelivery.`);
      onRefresh();
    } catch (err: any) {
      alert(`Error replaying DLQ: ${err.message}`);
    } finally {
      setReplayingAll(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* DLQ Header Strip */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-rose-950/20 border border-rose-500/30 p-5 rounded-2xl">
        <div className="space-y-1">
          <div className="flex items-center space-x-2">
            <ShieldAlert className="w-5 h-5 text-rose-400" />
            <h2 className="text-base font-bold text-white">Dead Letter Queue (DLQ)</h2>
            <span className="bg-rose-500/20 text-rose-400 text-xs font-mono font-bold px-2 py-0.5 rounded-full border border-rose-500/30">
              {failedDeliveries.length} Failed
            </span>
          </div>
          <p className="text-xs text-neutral-400">
            Webhooks that exhausted all 5 automated retry attempts (5s, 1m, 15m, 1h, 24h). Review downstream error codes or trigger batch redeliveries once customer servers recover.
          </p>
        </div>

        {failedDeliveries.length > 0 && (
          <button
            onClick={handleReplayAll}
            disabled={replayingAll}
            className="bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold px-4 py-2 rounded-lg transition-colors flex items-center space-x-2 shadow-sm whitespace-nowrap disabled:opacity-50"
          >
            <RotateCw className={`w-3.5 h-3.5 ${replayingAll ? 'animate-spin' : ''}`} />
            <span>Replay All Dead Letters</span>
          </button>
        )}
      </div>

      {/* DLQ Items */}
      {failedDeliveries.length === 0 ? (
        <div className="border border-neutral-800 bg-neutral-900/40 rounded-xl p-12 text-center">
          <div className="w-12 h-12 rounded-full bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center mx-auto text-emerald-400 mb-3">
            <CheckCircle2 className="w-6 h-6" />
          </div>
          <h4 className="text-sm font-semibold text-neutral-200">Dead Letter Queue Clean</h4>
          <p className="text-xs text-neutral-500 max-w-sm mx-auto mt-1">
            Zero abandoned webhooks. All outbound events succeeded or are still in active exponential retry cycles.
          </p>
        </div>
      ) : (
        <div className="border border-neutral-800 bg-neutral-900/50 rounded-xl overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-neutral-800 bg-neutral-900/80 text-[11px] font-semibold uppercase tracking-wider text-neutral-400 font-mono">
                  <th className="py-3 px-4">Event Type</th>
                  <th className="py-3 px-4">Destination Endpoint</th>
                  <th className="py-3 px-4">Last Status / Error</th>
                  <th className="py-3 px-4">Attempts</th>
                  <th className="py-3 px-4">Failed At</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-800/60 text-xs">
                {failedDeliveries.map((d) => (
                  <tr
                    key={d.id}
                    className="hover:bg-neutral-800/40 transition-colors cursor-pointer group"
                    onClick={() => onInspect(d)}
                  >
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
                    <td className="py-3 px-4">
                      <div className="flex items-center space-x-1.5 text-rose-400 font-mono">
                        <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                        <span className="truncate max-w-xs">{d.error_message || `HTTP ${d.http_status}`}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap font-mono text-neutral-400">
                      {d.attempt_number} of {d.max_attempts}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap text-neutral-400 font-mono text-[11px]">
                      {d.sent_at ? new Date(d.sent_at).toLocaleTimeString() : new Date(d.created_at).toLocaleTimeString()}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap text-right space-x-2" onClick={(e) => e.stopPropagation()}>
                      <button
                        onClick={() => onInspect(d)}
                        className="p-1.5 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded transition-colors inline-flex items-center"
                        title="Inspect payload and response body"
                      >
                        Inspect
                      </button>
                      <button
                        onClick={() => onRetry(d.id)}
                        className="bg-neutral-800 hover:bg-neutral-700 text-emerald-400 text-xs font-semibold px-2.5 py-1 rounded transition-colors inline-flex items-center space-x-1"
                        title="Replay this event"
                      >
                        <RotateCw className="w-3 h-3" />
                        <span>Replay</span>
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
