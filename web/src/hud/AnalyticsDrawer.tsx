import { useState, useEffect, useCallback } from 'react';
import {
  X,
  TrendingUp,
  Clock,
  Eye,
  Users,
  Compass,
  ArrowUpRight,
  BarChart2,
} from 'lucide-react';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
} from 'recharts';
import { Tenant } from '../types';

interface HourlyPoint {
  timestamp: number;
  label: string;
  pageviews: number;
  sessions: number;
}

interface TopItem {
  name: string;
  hits: number;
}

interface OverviewData {
  active_visitors: number;
  total_pageviews_24h: number;
  total_sessions_24h: number;
  avg_dwell_24h: number;
  hourly_points: HourlyPoint[];
}

interface TenantStatsData {
  tenant_id: string;
  range: string;
  points: HourlyPoint[];
  top_paths: TopItem[];
  top_referrers: TopItem[];
}

interface AnalyticsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  tenants: Tenant[];
}

export function AnalyticsDrawer({
  isOpen,
  onClose,
  tenants,
}: AnalyticsDrawerProps) {
  const [selectedTenantId, setSelectedTenantId] = useState<string>('overview');
  const [range, setRange] = useState<'24h' | '7d'>('24h');
  const [overview, setOverview] = useState<OverviewData | null>(null);
  const [tenantStats, setTenantStats] = useState<TenantStatsData | null>(null);
  const [loading, setLoading] = useState<boolean>(false);

  const fetchOverview = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/stats/overview');
      if (res.ok) {
        const data = await res.json();
        setOverview(data);
      }
    } catch {
      // Ignored
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTenantStats = useCallback(
    async (id: string, r: '24h' | '7d') => {
      setLoading(true);
      try {
        const res = await fetch(`/api/v1/stats/tenant/${id}?range=${r}`);
        if (res.ok) {
          const data = await res.json();
          setTenantStats(data);
        }
      } catch {
        // Ignored
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    if (!isOpen) return;
    if (selectedTenantId === 'overview') {
      fetchOverview();
    } else {
      fetchTenantStats(selectedTenantId, range);
    }
  }, [isOpen, selectedTenantId, range, fetchOverview, fetchTenantStats]);

  // Escape key handler
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    if (isOpen) {
      window.addEventListener('keydown', handleKeyDown);
    }
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/50 backdrop-blur-[2px] animate-fade-in">
      <div
        className="w-full max-w-xl h-full bg-[#09090b] border-l border-zinc-800 flex flex-col text-foreground overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Drawer Header */}
        <div className="p-4 border-b border-zinc-800 flex items-center justify-between bg-[#0c0c0f]">
          <div className="flex items-center space-x-2.5">
            <div className="w-7 h-7 rounded bg-zinc-900 border border-zinc-800 flex items-center justify-center text-accent">
              <BarChart2 className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-foreground">HUD Analytics</h2>
              <p className="text-[11px] text-foreground-muted">Traffic, Session & Dwell Metrics</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded hover:bg-surface-muted text-foreground-dim hover:text-foreground transition-colors"
            aria-label="Tutup Analytics"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Tenant Filter Selector */}
        <div className="p-3 border-b border-zinc-800 bg-[#09090b] flex items-center justify-between gap-2 overflow-x-auto">
          <div className="flex items-center space-x-1">
            <button
              onClick={() => setSelectedTenantId('overview')}
              className={`px-2.5 py-1 rounded text-xs transition-colors ${
                selectedTenantId === 'overview'
                  ? 'bg-accent text-white font-medium'
                  : 'text-foreground-muted hover:text-foreground hover:bg-zinc-900'
              }`}
            >
              Overview
            </button>
            {tenants.map((t) => (
              <button
                key={t.id}
                onClick={() => setSelectedTenantId(t.id)}
                className={`px-2.5 py-1 rounded text-xs transition-colors truncate max-w-[120px] ${
                  selectedTenantId === t.id
                    ? 'bg-accent text-white font-medium'
                    : 'text-foreground-muted hover:text-foreground hover:bg-zinc-900'
                }`}
              >
                {t.name}
              </button>
            ))}
          </div>

          {selectedTenantId !== 'overview' && (
            <div className="flex items-center bg-zinc-900 rounded border border-zinc-800 p-0.5 text-xs">
              <button
                onClick={() => setRange('24h')}
                className={`px-2 py-0.5 rounded ${
                  range === '24h' ? 'bg-zinc-800 text-white font-medium' : 'text-foreground-muted'
                }`}
              >
                24h
              </button>
              <button
                onClick={() => setRange('7d')}
                className={`px-2 py-0.5 rounded ${
                  range === '7d' ? 'bg-zinc-800 text-white font-medium' : 'text-foreground-muted'
                }`}
              >
                7d
              </button>
            </div>
          )}
        </div>

        {/* Drawer Body */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {loading ? (
            <div className="h-64 flex flex-col items-center justify-center text-xs text-foreground-muted">
              <span className="w-5 h-5 border-2 border-accent border-t-transparent rounded-full animate-spin mb-2" />
              <span>Memuat data analitik...</span>
            </div>
          ) : selectedTenantId === 'overview' && overview ? (
            <>
              {/* Stat Cards 4-Grid */}
              <div className="grid grid-cols-2 gap-3">
                <div className="p-3.5 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                  <div className="flex items-center space-x-2 text-foreground-muted text-xs mb-1">
                    <Users className="w-3.5 h-3.5 text-accent" />
                    <span>Live Visitors</span>
                  </div>
                  <div className="text-xl font-bold text-foreground">
                    {overview.active_visitors}
                  </div>
                </div>

                <div className="p-3.5 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                  <div className="flex items-center space-x-2 text-foreground-muted text-xs mb-1">
                    <Eye className="w-3.5 h-3.5 text-emerald-400" />
                    <span>Pageviews (24h)</span>
                  </div>
                  <div className="text-xl font-bold text-foreground">
                    {overview.total_pageviews_24h}
                  </div>
                </div>

                <div className="p-3.5 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                  <div className="flex items-center space-x-2 text-foreground-muted text-xs mb-1">
                    <TrendingUp className="w-3.5 h-3.5 text-blue-400" />
                    <span>Sessions (24h)</span>
                  </div>
                  <div className="text-xl font-bold text-foreground">
                    {overview.total_sessions_24h}
                  </div>
                </div>

                <div className="p-3.5 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                  <div className="flex items-center space-x-2 text-foreground-muted text-xs mb-1">
                    <Clock className="w-3.5 h-3.5 text-amber-400" />
                    <span>Avg Dwell (24h)</span>
                  </div>
                  <div className="text-xl font-bold text-foreground">
                    {overview.avg_dwell_24h}s
                  </div>
                </div>
              </div>

              {/* 24h Area Chart */}
              <div className="p-4 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                <h3 className="text-xs font-semibold text-foreground mb-3 flex items-center space-x-1.5">
                  <span>Aktivitas Pengunjung (24 Jam Terakhir)</span>
                </h3>
                {overview.hourly_points.length === 0 ? (
                  <div className="h-44 flex items-center justify-center text-xs text-foreground-dim">
                    Belum ada data traffic dalam 24 jam terakhir.
                  </div>
                ) : (
                  <div className="h-48 w-full">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={overview.hourly_points}>
                        <defs>
                          <linearGradient id="pvGrad" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
                            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <XAxis
                          dataKey="label"
                          stroke="#71717a"
                          fontSize={10}
                          tickLine={false}
                        />
                        <YAxis stroke="#71717a" fontSize={10} tickLine={false} />
                        <Tooltip
                          contentStyle={{
                            backgroundColor: '#121215',
                            borderColor: '#27272a',
                            borderRadius: '6px',
                            fontSize: '11px',
                          }}
                        />
                        <Area
                          type="monotone"
                          dataKey="pageviews"
                          stroke="#3b82f6"
                          fillOpacity={1}
                          fill="url(#pvGrad)"
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </div>
            </>
          ) : tenantStats ? (
            <>
              {/* Tenant Trend Chart */}
              <div className="p-4 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                <h3 className="text-xs font-semibold text-foreground mb-3">
                  Aktivitas Traffic ({range})
                </h3>
                {tenantStats.points.length === 0 ? (
                  <div className="h-44 flex items-center justify-center text-xs text-foreground-dim">
                    Belum ada rekaman traffic untuk rentang ini.
                  </div>
                ) : (
                  <div className="h-48 w-full">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={tenantStats.points}>
                        <defs>
                          <linearGradient id="tenantPvGrad" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
                            <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <XAxis
                          dataKey="label"
                          stroke="#71717a"
                          fontSize={10}
                          tickLine={false}
                        />
                        <YAxis stroke="#71717a" fontSize={10} tickLine={false} />
                        <Tooltip
                          contentStyle={{
                            backgroundColor: '#0c0c0f',
                            borderColor: '#27272a',
                            borderRadius: '6px',
                            fontSize: '11px',
                          }}
                        />
                        <Area
                          type="monotone"
                          dataKey="pageviews"
                          stroke="#10b981"
                          fillOpacity={1}
                          fill="url(#tenantPvGrad)"
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </div>

              {/* Top Paths & Top Referrers Tables */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Top Paths */}
                <div className="p-3.5 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                  <h4 className="text-xs font-semibold text-foreground mb-2 flex items-center space-x-1.5">
                    <Compass className="w-3.5 h-3.5 text-accent" />
                    <span>Top Halaman</span>
                  </h4>
                  {tenantStats.top_paths.length === 0 ? (
                    <div className="text-xs text-foreground-dim py-4 text-center">
                      Belum ada data path.
                    </div>
                  ) : (
                    <div className="space-y-1.5 text-xs">
                      {tenantStats.top_paths.map((p, i) => (
                        <div
                          key={i}
                          className="flex items-center justify-between py-1 border-b border-zinc-800/60"
                        >
                          <span className="truncate max-w-[160px] text-foreground-muted font-mono text-[11px]">
                            {p.name}
                          </span>
                          <span className="font-semibold text-foreground">{p.hits}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Top Referrers */}
                <div className="p-3.5 rounded-lg bg-[#0c0c0f] border border-zinc-800">
                  <h4 className="text-xs font-semibold text-foreground mb-2 flex items-center space-x-1.5">
                    <ArrowUpRight className="w-3.5 h-3.5 text-emerald-400" />
                    <span>Top Rujukan</span>
                  </h4>
                  {tenantStats.top_referrers.length === 0 ? (
                    <div className="text-xs text-foreground-dim py-4 text-center">
                      Belum ada rujukan eksternal.
                    </div>
                  ) : (
                    <div className="space-y-1.5 text-xs">
                      {tenantStats.top_referrers.map((r, i) => (
                        <div
                          key={i}
                          className="flex items-center justify-between py-1 border-b border-zinc-800/60"
                        >
                          <span className="truncate max-w-[160px] text-foreground-muted text-[11px]">
                            {r.name}
                          </span>
                          <span className="font-semibold text-foreground">{r.hits}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </>
          ) : null}
        </div>
      </div>
    </div>
  );
}
