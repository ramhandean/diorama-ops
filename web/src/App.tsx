import { useState, useMemo, useEffect, lazy, Suspense } from 'react';
import { Box, Layers, ShieldCheck, Users, Radio, BarChart2 } from 'lucide-react';
import { useDioramaSocket } from './hooks/useDioramaSocket';
import { Fallback2D } from './fallback2d/Fallback2D';
import { AdminDashboard } from './admin/AdminDashboard';
import { Logo } from './components/Logo';
import { FloatingTelemetryHUD } from './hud/FloatingTelemetryHUD';

// Lazy load Three.js Canvas and Heavy Modals
const Scene3DCanvas = lazy(() => import('./scene/Scene3DCanvas'));
const TenantDetailModal = lazy(() =>
  import('./hud/TenantDetailModal').then((m) => ({ default: m.TenantDetailModal }))
);
const AnalyticsDrawer = lazy(() =>
  import('./hud/AnalyticsDrawer').then((m) => ({ default: m.AnalyticsDrawer }))
);

function checkWebGL(): boolean {
  try {
    const canvas = document.createElement('canvas');
    return !!(
      window.WebGLRenderingContext &&
      (canvas.getContext('webgl') || canvas.getContext('experimental-webgl'))
    );
  } catch {
    return false;
  }
}

export default function App() {
  const {
    tenants,
    avatars,
    connected,
    selectedTenant,
    setSelectedTenant,
    refetch,
  } = useDioramaSocket();

  const webGLSupported = useMemo(() => checkWebGL(), []);

  const [viewMode, setViewMode] = useState<'3d' | '2d' | 'admin'>(() => {
    if (window.location.pathname.startsWith('/admin') || window.location.hash === '#admin') {
      return 'admin';
    }
    return checkWebGL() ? '3d' : '2d';
  });

  useEffect(() => {
    if (viewMode === '3d' || viewMode === '2d') {
      refetch();
    }
  }, [viewMode, refetch]);

  const [analyticsOpen, setAnalyticsOpen] = useState<boolean>(false);

  const totalActiveVisitors = useMemo(() => {
    return tenants.reduce((acc, t) => acc + t.active, 0);
  }, [tenants]);

  return (
    <div className="relative w-full h-[100dvh] flex flex-col bg-background text-foreground select-none overflow-hidden font-sans">
      {/* Top Header Bar - seamlessly docked to #09090b */}
      <header className="absolute top-0 left-0 right-0 z-20 flex items-center justify-between px-4 py-2.5 bg-[#09090b]/85 backdrop-blur-sm border-b border-zinc-800/80">
        <div className="flex items-center space-x-3">
          <div className="p-1 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-center">
            <Logo size={24} />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <h1 className="text-sm font-semibold tracking-wide text-foreground">DioramaOps</h1>
              <span className="text-[10px] px-1.5 py-0.2 rounded bg-zinc-800/80 text-foreground-dim font-mono">
                v0.1
              </span>
            </div>
            <p className="text-[11px] text-foreground-muted hidden sm:block">
              Interactive 3D Infrastructure & Telemetry Hub
            </p>
          </div>
        </div>

        {/* Status Indicators & View Mode Toggle */}
        <div className="flex items-center space-x-2 sm:space-x-3">
          {/* Active Visitors Counter */}
          <div className="flex items-center space-x-1.5 px-2.5 py-1 rounded bg-zinc-900/70 border border-zinc-800 text-xs text-foreground">
            <Users className="w-3.5 h-3.5 text-accent" />
            <span>
              <strong className="font-semibold">{totalActiveVisitors}</strong> live
            </span>
          </div>

          {/* Hub Realtime Socket Connection Status */}
          <div className="flex items-center space-x-1.5 px-2.5 py-1 rounded bg-zinc-900/70 border border-zinc-800 text-xs text-foreground-muted">
            <span
              className={`w-2 h-2 rounded-full ${
                connected ? 'bg-emerald-500 animate-pulse' : 'bg-amber-500'
              }`}
            />
            <span className="hidden sm:inline">
              {connected ? 'Terkoneksi' : 'Menghubungkan...'}
            </span>
          </div>

          {/* Analytics Drawer Button */}
          <button
            onClick={() => setAnalyticsOpen(true)}
            className="flex items-center space-x-1.5 px-2.5 py-1 rounded bg-zinc-900/70 hover:bg-zinc-800 border border-zinc-800 text-xs text-foreground transition-colors"
            aria-label="Buka Analytics"
          >
            <BarChart2 className="w-3.5 h-3.5 text-accent" />
            <span className="hidden sm:inline">Analytics</span>
          </button>

          {/* Projects Portal Button */}
          <button
            onClick={() => setViewMode(viewMode === 'admin' ? (webGLSupported ? '3d' : '2d') : 'admin')}
            className={`flex items-center space-x-1.5 px-2.5 py-1 rounded border text-xs transition-colors ${
              viewMode === 'admin'
                ? 'bg-accent text-white font-medium border-accent'
                : 'bg-zinc-900/70 hover:bg-zinc-800 border-zinc-800 text-foreground'
            }`}
            aria-label="Portal Projects"
            title="Kelola Project & Konfigurasi"
          >
            <Layers className="w-3.5 h-3.5 text-accent" />
            <span className="hidden sm:inline">Projects</span>
          </button>

          {/* Toggle 3D / 2D View */}
          {webGLSupported && viewMode !== 'admin' && (
            <div className="flex items-center bg-zinc-900/80 rounded border border-zinc-800 p-0.5">
              <button
                onClick={() => setViewMode('3d')}
                className={`flex items-center space-x-1 px-2.5 py-1 rounded text-xs transition-colors ${
                  viewMode === '3d'
                    ? 'bg-zinc-800 text-white font-medium border border-zinc-700/60'
                    : 'text-foreground-muted hover:text-foreground'
                }`}
                aria-label="Tampilan 3D"
              >
                <Radio className="w-3 h-3 text-accent" />
                <span className="hidden sm:inline">3D</span>
              </button>
              <button
                onClick={() => setViewMode('2d')}
                className={`flex items-center space-x-1 px-2.5 py-1 rounded text-xs transition-colors ${
                  viewMode === '2d'
                    ? 'bg-zinc-800 text-white font-medium border border-zinc-700/60'
                    : 'text-foreground-muted hover:text-foreground'
                }`}
                aria-label="Tampilan 2D"
              >
                <Layers className="w-3 h-3 text-accent" />
                <span className="hidden sm:inline">2D</span>
              </button>
            </div>
          )}
        </div>
      </header>

      {/* Main View Area */}
      <main className="w-full h-full relative pt-14">
        {viewMode === 'admin' ? (
          <AdminDashboard
            onBackToScene={() => setViewMode(webGLSupported ? '3d' : '2d')}
            onTenantsChanged={refetch}
          />
        ) : viewMode === '3d' && webGLSupported ? (
          <Suspense
            fallback={
              <div className="w-full h-full flex flex-col items-center justify-center text-xs text-foreground-muted">
                <Box className="w-6 h-6 mb-2 text-accent animate-spin" />
                <span>Memuat Diorama 3D...</span>
              </div>
            }
          >
            <Scene3DCanvas
              tenants={tenants}
              avatars={avatars}
              onSelectTenant={(t) => setSelectedTenant(t)}
            />
            {/* Floating Telemetry HUD in 3D scene */}
            <FloatingTelemetryHUD
              tenants={tenants}
              onSelectTenant={(t) => setSelectedTenant(t)}
            />
          </Suspense>
        ) : (
          <Fallback2D
            tenants={tenants}
            onSelectTenant={(t) => setSelectedTenant(t)}
            onOpenCreate={() => setViewMode('admin')}
          />
        )}
      </main>

      {/* Footer Privacy Note */}
      <footer className="absolute bottom-3 left-4 z-10 flex items-center space-x-2 text-[11px] text-foreground-dim pointer-events-none">
        <ShieldCheck className="w-3.5 h-3.5 text-accent" />
        <span>Privasi: Tanpa cookie, tanpa log IP mentah, rotasi salt harian</span>
      </footer>

      {/* Detail Modal */}
      {selectedTenant && (
        <Suspense fallback={null}>
          <TenantDetailModal
            tenant={selectedTenant}
            onClose={() => setSelectedTenant(null)}
          />
        </Suspense>
      )}

      {/* Analytics Drawer */}
      {analyticsOpen && (
        <Suspense fallback={null}>
          <AnalyticsDrawer
            isOpen={analyticsOpen}
            onClose={() => setAnalyticsOpen(false)}
            tenants={tenants}
          />
        </Suspense>
      )}
    </div>
  );
}
