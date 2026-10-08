import { Tenant } from '../types';
import { TenantAvatar } from '../components/TenantAvatar';
import { X, ExternalLink, Users, Activity, Globe } from 'lucide-react';

interface TenantDetailModalProps {
  tenant: Tenant | null;
  onClose: () => void;
}

export function TenantDetailModal({ tenant, onClose }: TenantDetailModalProps) {
  if (!tenant) return null;

  const accentColor = tenant.accent || '#3b82f6';

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-[2px] animate-fade-in"
      onClick={onClose}
    >
      <div
        className="relative w-full max-w-md p-6 rounded-lg bg-[#0c0c0f] border border-zinc-800 text-foreground"
        onClick={(e) => e.stopPropagation()}
        style={{
          borderTopColor: accentColor,
          borderTopWidth: '2px',
        }}
      >
        <button
          onClick={onClose}
          className="absolute top-4 right-4 p-2 rounded-md hover:bg-surface-subtle text-foreground-dim hover:text-foreground transition-colors"
          aria-label="Tutup panel detail"
        >
          <X className="w-4 h-4" />
        </button>

        <div className="flex items-center space-x-3 mb-4">
          <TenantAvatar
            id={tenant.id}
            name={tenant.name}
            accent={accentColor}
            logoPath={tenant.logo_path}
            useFavicon={tenant.use_favicon}
            size="lg"
          />
          <div>
            <h2 className="text-base font-semibold text-foreground">{tenant.name}</h2>
            <div className="flex items-center space-x-2 mt-0.5 text-xs text-foreground-muted">
              <span
                className={`w-2 h-2 rounded-full ${
                  tenant.status === 'up'
                    ? 'bg-emerald-500'
                    : tenant.status === 'degraded'
                    ? 'bg-amber-500'
                    : 'bg-rose-500'
                }`}
              />
              <span className="capitalize">{tenant.status}</span>
            </div>
          </div>
        </div>

        <div className="space-y-3 py-3 border-y border-border text-sm">
          <div className="flex items-center justify-between">
            <span className="flex items-center space-x-2 text-foreground-muted">
              <Globe className="w-4 h-4 text-foreground-dim" />
              <span>URL Situs</span>
            </span>
            <a
              href={tenant.url}
              target="_blank"
              rel="noreferrer"
              className="flex items-center space-x-1 text-accent hover:underline truncate max-w-[200px]"
            >
              <span>{tenant.url.replace(/^https?:\/\//, '')}</span>
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          </div>

          <div className="flex items-center justify-between">
            <span className="flex items-center space-x-2 text-foreground-muted">
              <Users className="w-4 h-4 text-foreground-dim" />
              <span>Pengunjung Aktif</span>
            </span>
            <span className="font-semibold text-foreground">
              {tenant.active} live
            </span>
          </div>

          <div className="flex items-center justify-between">
            <span className="flex items-center space-x-2 text-foreground-muted">
              <Activity className="w-4 h-4 text-foreground-dim" />
              <span>Kesehatan Sistem</span>
            </span>
            <span className="text-xs px-2 py-0.5 rounded bg-surface-subtle border border-border uppercase tracking-wide">
              {tenant.status}
            </span>
          </div>
        </div>

        <div className="mt-5 flex justify-end">
          <a
            href={tenant.url}
            target="_blank"
            rel="noreferrer"
            className="flex items-center justify-center space-x-2 w-full px-4 py-2 rounded bg-accent hover:bg-accent-hover text-white text-sm font-medium transition-colors"
          >
            <span>Kunjungi Situs</span>
            <ExternalLink className="w-4 h-4" />
          </a>
        </div>
      </div>
    </div>
  );
}
