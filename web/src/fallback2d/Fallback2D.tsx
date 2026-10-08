import { Tenant } from '../types';
import { TenantAvatar } from '../components/TenantAvatar';
import { ExternalLink, Users, Activity, Plus } from 'lucide-react';

interface Fallback2DProps {
  tenants: Tenant[];
  onSelectTenant: (t: Tenant) => void;
  onOpenCreate?: () => void;
}

export function Fallback2D({ tenants = [], onSelectTenant, onOpenCreate }: Fallback2DProps) {
  if (!tenants || tenants.length === 0) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center p-6 text-center text-foreground-muted space-y-4">
        <div className="w-12 h-12 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center text-blue-400">
          <Activity className="w-6 h-6 text-blue-400" />
        </div>
        <div className="space-y-1 max-w-sm">
          <p className="text-sm font-medium text-zinc-200">Belum Ada Tenant Terdaftar</p>
          <p className="text-xs text-zinc-400 leading-relaxed">
            Daftarkan website atau layanan pertamamu untuk mulai memonitor uptime & analitik secara real-time.
          </p>
        </div>
        {onOpenCreate && (
          <button
            onClick={onOpenCreate}
            className="flex items-center space-x-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" />
            <span>Tambah Project / Booth Pertamamu</span>
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="w-full h-full overflow-y-auto p-4 sm:p-6 lg:p-8">
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {tenants.map((t) => {
            const isUp = t.status !== 'down';
            return (
              <div
                key={t.id}
                onClick={() => onSelectTenant(t)}
                className="bg-zinc-900/60 hover:bg-zinc-900/90 border border-zinc-800/80 hover:border-zinc-700/80 rounded-xl p-4 transition-all duration-150 cursor-pointer flex flex-col justify-between space-y-3 group shadow-sm hover:shadow-md"
              >
                {/* Header: Logo, Name, and Status indicator */}
                <div className="flex items-start justify-between">
                  <div className="flex items-center space-x-3 min-w-0">
                    <TenantAvatar
                      id={t.id}
                      name={t.name}
                      accent={t.accent}
                      logoPath={t.logo_path}
                      useFavicon={t.use_favicon}
                      size="md"
                    />
                    <div className="min-w-0">
                      <h3 className="text-sm font-semibold text-foreground truncate group-hover:text-accent transition-colors">
                        {t.name}
                      </h3>
                      <p className="text-xs text-foreground-dim truncate">
                        {t.url ? new URL(t.url).hostname : 'No URL'}
                      </p>
                    </div>
                  </div>

                  <span
                    className={`w-2.5 h-2.5 rounded-full mt-1.5 shrink-0 ${
                      isUp ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.5)]'
                    }`}
                  />
                </div>

                {/* Metrics / Telemetry details */}
                <div className="flex items-center justify-between pt-2 border-t border-zinc-800/50 text-xs text-foreground-muted">
                  <div className="flex items-center space-x-1">
                    <Users className="w-3.5 h-3.5 text-foreground-dim" />
                    <span>
                      <strong className="text-foreground font-medium">{t.active}</strong> live
                    </span>
                  </div>

                  <div className="flex items-center space-x-2">
                    <span className="text-[11px] font-mono capitalize">
                      {t.status}
                    </span>
                    {t.url && (
                      <a
                        href={t.url}
                        target="_blank"
                        rel="noreferrer"
                        onClick={(e) => e.stopPropagation()}
                        className="text-foreground-dim hover:text-foreground p-0.5 rounded transition-colors"
                        aria-label={`Buka ${t.name}`}
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </a>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
