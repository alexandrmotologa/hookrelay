import React from 'react';
import { Send, Shield, AlertTriangle, Layers, Radio, RefreshCw, Sparkles, Terminal, Activity } from 'lucide-react';
import { StatsSummary } from '../types';

export type ActiveTab = 'deliveries' | 'endpoints' | 'dlq' | 'simulator' | 'portal' | 'mock' | 'debugger';

interface HeaderProps {
  activeTab: ActiveTab;
  setActiveTab: (tab: ActiveTab) => void;
  stats: StatsSummary | null;
  autoRefresh: boolean;
  setAutoRefresh: (auto: boolean) => void;
  onRefresh: () => void;
  onOpenSimulator: () => void;
  sseConnected: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  activeTab,
  setActiveTab,
  stats,
  autoRefresh,
  setAutoRefresh,
  onRefresh,
  onOpenSimulator,
  sseConnected,
}) => {
  return (
    <header className="border-b border-neutral-800 bg-neutral-900/90 backdrop-blur sticky top-0 z-30">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16 gap-3">
          {/* Logo & Brand (compact & sharp) */}
          <div
            className="flex items-center space-x-2.5 cursor-pointer flex-shrink-0"
            onClick={() => setActiveTab('deliveries')}
          >
            <div className="w-8 h-8 sm:w-9 sm:h-9 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 flex-shrink-0">
              <Radio className="w-4 h-4 sm:w-5 sm:h-5" />
            </div>
            <div className="flex items-center space-x-2">
              <span className="font-bold text-lg tracking-tight text-white">HookRelay</span>
              <span className="text-[10px] font-mono uppercase bg-neutral-800 border border-neutral-700 px-1.5 py-0.5 rounded text-neutral-400">
                v1.1
              </span>
            </div>
          </div>

          {/* Navigation Tabs (spacious, never overlapping) */}
          <nav className="flex items-center space-x-1 sm:space-x-1.5 overflow-x-auto no-scrollbar py-1">
            <button
              onClick={() => setActiveTab('deliveries')}
              title="Deliveries Stream"
              className={`px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-medium rounded-md transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0 ${
                activeTab === 'deliveries'
                  ? 'bg-neutral-800 text-white border border-neutral-700 shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              <Layers className="w-4 h-4 text-emerald-400" />
              <span>Deliveries</span>
            </button>

            <button
              onClick={() => setActiveTab('endpoints')}
              title="Destination Endpoints"
              className={`px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-medium rounded-md transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0 ${
                activeTab === 'endpoints'
                  ? 'bg-neutral-800 text-white border border-neutral-700 shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              <Shield className="w-4 h-4 text-sky-400" />
              <span>Endpoints</span>
              {stats && stats.suspended_endpoints > 0 && (
                <span className="bg-rose-500/20 text-rose-400 text-[10px] font-bold px-1.5 py-0.2 rounded-full border border-rose-500/30">
                  {stats.suspended_endpoints}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('dlq')}
              title="Dead Letter Queue"
              className={`px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-medium rounded-md transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0 ${
                activeTab === 'dlq'
                  ? 'bg-neutral-800 text-white border border-neutral-700 shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              <AlertTriangle className="w-4 h-4 text-rose-400" />
              <span>Dead Letter</span>
              {stats && stats.failed_count > 0 && (
                <span className="bg-amber-500/20 text-amber-400 text-[10px] font-bold px-1.5 py-0.2 rounded-full border border-amber-500/30">
                  {stats.failed_count}
                </span>
              )}
            </button>

            <button
              onClick={() => setActiveTab('mock')}
              title="Built-in Mock Receiver & Chaos Simulator"
              className={`px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-medium rounded-md transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0 ${
                activeTab === 'mock'
                  ? 'bg-neutral-800 text-white border border-neutral-700 shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              <Radio className="w-4 h-4 text-cyan-400" />
              <span>Mock Receiver</span>
            </button>

            <button
              onClick={() => setActiveTab('debugger')}
              title="Standard Webhooks HMAC-SHA256 Signature Debugger"
              className={`px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-medium rounded-md transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0 ${
                activeTab === 'debugger'
                  ? 'bg-neutral-800 text-white border border-neutral-700 shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              <Terminal className="w-4 h-4 text-emerald-400" />
              <span>Debugger</span>
            </button>

            <button
              onClick={() => setActiveTab('portal')}
              title="Customer Self-Serve Delivery Portal"
              className={`px-2.5 sm:px-3 py-1.5 text-xs sm:text-sm font-medium rounded-md transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0 ${
                activeTab === 'portal'
                  ? 'bg-neutral-800 text-white border border-neutral-700 shadow-sm'
                  : 'text-neutral-400 hover:text-white hover:bg-neutral-800/50'
              }`}
            >
              <Sparkles className="w-4 h-4 text-amber-400" />
              <span>Customer Portal</span>
            </button>
          </nav>

          {/* Quick Actions & Live indicator (rigid flex-shrink-0, clean padding) */}
          <div className="flex items-center space-x-2 sm:space-x-2.5 flex-shrink-0">
            {/* Real-time SSE indicator */}
            <div
              className={`flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-[11px] font-mono border whitespace-nowrap flex-shrink-0 ${
                sseConnected
                  ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                  : 'bg-neutral-900 border-neutral-800 text-neutral-500'
              }`}
              title={sseConnected ? 'Server-Sent Events (SSE) active (<10ms)' : 'Polling mode active'}
            >
              <Activity className={`w-3 h-3 flex-shrink-0 ${sseConnected ? 'animate-pulse text-emerald-400' : 'text-neutral-500'}`} />
              <span className="hidden sm:inline">{sseConnected ? 'SSE Live' : 'Polling'}</span>
            </div>

            <button
              onClick={onOpenSimulator}
              title="Publish test webhook"
              className="bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold px-3 py-1.5 rounded-md shadow-sm transition-colors flex items-center space-x-1.5 whitespace-nowrap flex-shrink-0"
            >
              <Send className="w-3.5 h-3.5 flex-shrink-0" />
              <span>Publish</span>
            </button>

            <button
              onClick={() => {
                onRefresh();
                setAutoRefresh(!autoRefresh);
              }}
              title={autoRefresh ? 'Live Polling Active. Click to pause.' : 'Polling Paused. Click to refresh & resume.'}
              className={`p-1.5 sm:p-2 rounded-md border text-xs transition-colors flex items-center justify-center flex-shrink-0 ${
                autoRefresh
                  ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                  : 'bg-neutral-900 border-neutral-800 text-neutral-500 hover:text-neutral-300'
              }`}
            >
              <RefreshCw className={`w-3.5 h-3.5 flex-shrink-0 ${autoRefresh ? 'animate-spin' : ''}`} style={{ animationDuration: '4s' }} />
            </button>
          </div>
        </div>
      </div>
    </header>
  );
};
