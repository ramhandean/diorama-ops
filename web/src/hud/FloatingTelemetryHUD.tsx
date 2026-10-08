import React, { useState } from 'react';
import { Activity, ChevronDown, ChevronUp, Radio, Users } from 'lucide-react';
import { Tenant } from '../types';

interface FloatingTelemetryHUDProps {
  tenants: Tenant[];
  onSelectTenant: (tenant: Tenant) => void;
}

export const FloatingTelemetryHUD: React.FC<FloatingTelemetryHUDProps> = ({
  tenants,
  onSelectTenant,
}) => {
  const [collapsed, setCollapsed] = useState(false);

  if (!tenants || tenants.length === 0) return null;

  // Take up to 4 active tenants for quick-glance status
  const topTenants = tenants.slice(0, 4);

  return (
    <div className="absolute right-3 sm:right-6 bottom-10 z-20 w-64 sm:w-72 flex flex-col space-y-1.5 pointer-events-auto">
      {/* Header bar */}
      <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-[#0f1015]/95 border border-zinc-800 backdrop-blur-md shadow-lg">
        <div className="flex items-center space-x-2">
          <Activity className="w-3.5 h-3.5 text-blue-400" />
          <span className="text-xs font-semibold text-white tracking-wide">
            Live Telemetry HUD
          </span>
        </div>
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="p-1 rounded text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors focus:outline-none focus:ring-1 focus:ring-blue-500"
          aria-label={collapsed ? 'Perluas HUD' : 'Ciutkan HUD'}
          title={collapsed ? 'Perluas' : 'Ciutkan'}
        >
          {collapsed ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
        </button>
      </div>

      {/* Booth Cards List */}
      {!collapsed && (
        <div className="space-y-1.5 animate-in fade-in duration-150">
          {topTenants.map((t) => {
            const isUp = t.status !== 'down' && t.status !== 'degraded';
            const statusLabel = t.status === 'down' ? 'DOWN' : t.status === 'degraded' ? 'DEGRADED' : '99.99% UP';
            const statusColor = t.status === 'down' ? 'text-red-400' : t.status === 'degraded' ? 'text-amber-400' : 'text-emerald-400';

            return (
              <button
                key={t.id}
                onClick={() => onSelectTenant(t)}
                className="w-full text-left p-2.5 rounded-lg bg-[#0f1015]/90 hover:bg-[#141622] border border-zinc-800/90 hover:border-blue-500/50 backdrop-blur-md transition-all duration-150 flex items-center justify-between group focus:outline-none focus:ring-1 focus:ring-blue-500 shadow-md"
              >
                <div className="flex items-center space-x-2.5 min-w-0">
                  <div
                    className="w-2.5 h-2.5 rounded-full shrink-0"
                    style={{ backgroundColor: t.accent || '#38bdf8' }}
                  />
                  <div className="flex flex-col min-w-0">
                    <span className="text-xs font-medium text-zinc-200 group-hover:text-white truncate">
                      {t.name}
                    </span>
                    <div className="flex items-center space-x-1.5 text-[10px] text-zinc-400 font-mono">
                      <Radio className="w-2.5 h-2.5 text-zinc-400" />
                      <span>{t.status.toUpperCase()}</span>
                      <span>•</span>
                      <Users className="w-2.5 h-2.5 text-zinc-400" />
                      <span>{t.active} live</span>
                    </div>
                  </div>
                </div>

                <div className="flex flex-col items-end shrink-0 pl-2">
                  <span className={`text-[11px] font-bold font-mono ${statusColor}`}>
                    {statusLabel}
                  </span>
                  <span className="text-[9px] text-zinc-400 font-mono">
                    {isUp ? 'Operational' : 'Incident'}
                  </span>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};
