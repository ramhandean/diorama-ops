import React, { useState, useEffect, useCallback, useId, useRef } from 'react';
import {
  Layers,
  Plus,
  Trash2,
  Edit2,
  Upload,
  Download,
  Copy,
  Check,
  Globe,
  X,
  Bot,
  ExternalLink,
  ChevronLeft,
  UploadCloud,
} from 'lucide-react';
import { Tenant } from '../types';
import { TenantAvatar } from '../components/TenantAvatar';
import { ToastContainer, type ToastMessage } from '../components/Toast';
import { ConfirmModal } from '../components/ConfirmModal';

interface AdminDashboardProps {
  onBackToScene: () => void;
  onTenantsChanged?: () => void;
}

interface TenantFormData {
  id: string;
  name: string;
  url: string;
  accent: string;
  origins: string;
  position_x: string;
  position_y: string;
  position_z: string;
  kuma_url: string;
  health_url: string;
  use_favicon: boolean;
}

const emptyForm: TenantFormData = {
  id: '',
  name: '',
  url: '',
  accent: '#3b82f6',
  origins: '',
  position_x: '',
  position_y: '',
  position_z: '',
  kuma_url: '',
  health_url: '',
  use_favicon: true,
};

const NEON_PRESETS = [
  { label: 'Cyan', hex: '#06b6d4' },
  { label: 'Blue', hex: '#3b82f6' },
  { label: 'Purple', hex: '#8b5cf6' },
  { label: 'Emerald', hex: '#10b981' },
  { label: 'Amber', hex: '#f59e0b' },
  { label: 'Rose', hex: '#f43f5e' },
];

export function AdminDashboard({ onBackToScene, onTenantsChanged }: AdminDashboardProps) {
  const secretInputId = useId();
  const fileInputId = useId();
  const [adminToken, setAdminToken] = useState<string>(() => {
    return sessionStorage.getItem('diorama_admin_secret') || '';
  });
  const [tokenInput, setTokenInput] = useState<string>('');
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string>('');

  // Custom Toast state
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const showToast = useCallback((message: string, type: 'info' | 'success' | 'warning' | 'error' = 'info') => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((prev) => [...prev, { id, message, type }]);
  }, []);
  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // Custom Delete Confirm Modal state
  const [deleteConfirm, setDeleteConfirm] = useState<{
    isOpen: boolean;
    tenantId: string;
    tenantName: string;
  }>({
    isOpen: false,
    tenantId: '',
    tenantName: '',
  });

  // Modal states
  const [editModalOpen, setEditModalOpen] = useState<boolean>(false);
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [formData, setFormData] = useState<TenantFormData>(emptyForm);

  const [uploadModalOpen, setUploadModalOpen] = useState<boolean>(false);
  const [uploadingTenant, setUploadingTenant] = useState<Tenant | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [filePreview, setFilePreview] = useState<string | null>(null);
  const [uploadError, setUploadError] = useState<string>('');
  const [isDragging, setIsDragging] = useState<boolean>(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  function closeUploadModal() {
    setUploadModalOpen(false);
    setSelectedFile(null);
    setFilePreview(null);
    setUploadingTenant(null);
    setUploadError('');
    setIsDragging(false);
  }

  const [promptCopiedId, setPromptCopiedId] = useState<string | null>(null);
  const [keyCopiedId, setKeyCopiedId] = useState<string | null>(null);

  // Fetch admin tenants list
  const fetchAdminTenants = useCallback(async (token: string) => {
    if (!token) return;
    setLoading(true);
    setErrorMsg('');
    try {
      const res = await fetch('/api/v1/admin/tenants', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        if (res.status === 401) {
          setErrorMsg('Kredensial tidak valid. Silakan periksa ADMIN_SECRET.');
          setAdminToken('');
          sessionStorage.removeItem('diorama_admin_secret');
        } else {
          setErrorMsg('Gagal memuat daftar tenant.');
        }
        return;
      }
      const data = await res.json();
      setTenants(Array.isArray(data) ? data : []);
    } catch {
      setErrorMsg('Koneksi ke server gagal.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (adminToken) {
      fetchAdminTenants(adminToken);
    }
  }, [adminToken, fetchAdminTenants]);

  function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    if (!tokenInput.trim()) return;
    sessionStorage.setItem('diorama_admin_secret', tokenInput.trim());
    setAdminToken(tokenInput.trim());
  }

  function handleLogout() {
    sessionStorage.removeItem('diorama_admin_secret');
    setAdminToken('');
    setTenants([]);
  }

  // Open Create Modal
  function handleOpenCreate() {
    if (tenants.length >= 1) {
      showToast('Versi Self-Hosted dibatasi 1 project per instance. Untuk project tambahan, jalankan instance baru.', 'warning');
      return;
    }
    setFormData(emptyForm);
    setIsEditing(false);
    setEditModalOpen(true);
  }

  // Open Edit Modal
  function handleOpenEdit(t: Tenant) {
    const rawOrigins = Array.isArray(t['origins' as keyof Tenant])
      ? (t['origins' as keyof Tenant] as unknown as string[]).join(', ')
      : '';
    setFormData({
      id: t.id,
      name: t.name,
      url: t.url,
      accent: t.accent || '#3b82f6',
      origins: rawOrigins,
      position_x: t.position_x !== undefined && t.position_x !== null ? String(t.position_x) : '',
      position_y: t.position_y !== undefined && t.position_y !== null ? String(t.position_y) : '',
      position_z: t.position_z !== undefined && t.position_z !== null ? String(t.position_z) : '',
      kuma_url: (t['kuma_url' as keyof Tenant] as unknown as string) || '',
      health_url: (t['health_url' as keyof Tenant] as unknown as string) || '',
      use_favicon: t.use_favicon,
    });
    setIsEditing(true);
    setEditModalOpen(true);
  }

  // Save Tenant (Create or Update)
  async function handleSaveTenant(e: React.FormEvent) {
    e.preventDefault();
    const originsList = formData.origins
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);

    const payload: Record<string, unknown> = {
      id: formData.id.trim(),
      name: formData.name.trim(),
      url: formData.url.trim(),
      accent: formData.accent.trim(),
      origins: originsList,
      use_favicon: formData.use_favicon,
    };

    if (formData.position_x !== '') payload.position_x = parseFloat(formData.position_x);
    if (formData.position_y !== '') payload.position_y = parseFloat(formData.position_y);
    if (formData.position_z !== '') payload.position_z = parseFloat(formData.position_z);
    if (formData.kuma_url !== '') payload.kuma_url = formData.kuma_url.trim();
    if (formData.health_url !== '') payload.health_url = formData.health_url.trim();

    const url = isEditing
      ? `/api/v1/admin/tenants/${formData.id}`
      : '/api/v1/admin/tenants';
    const method = isEditing ? 'PUT' : 'POST';

    try {
      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adminToken}`,
        },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        showToast(errData.error || 'Operasi gagal disimpan.', 'error');
        return;
      }

      setEditModalOpen(false);
      fetchAdminTenants(adminToken);
      showToast(isEditing ? 'Tenant berhasil diperbarui.' : 'Tenant baru berhasil ditambahkan.', 'success');
      onTenantsChanged?.();
    } catch {
      showToast('Terjadi kesalahan jaringan.', 'error');
    }
  }

  // Delete Tenant Trigger
  function handleDelete(id: string) {
    const target = tenants.find((t) => t.id === id);
    setDeleteConfirm({
      isOpen: true,
      tenantId: id,
      tenantName: target ? target.name : id,
    });
  }

  // Execute Deletion
  async function executeDelete(id: string) {
    try {
      const res = await fetch(`/api/v1/admin/tenants/${encodeURIComponent(id)}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${adminToken}` },
      });
      if (res.ok) {
        showToast(`Tenant '${id}' berhasil dihapus.`, 'success');
        fetchAdminTenants(adminToken);
        onTenantsChanged?.();
      } else {
        showToast('Gagal menghapus tenant.', 'error');
      }
    } catch {
      showToast('Terjadi kesalahan jaringan.', 'error');
    } finally {
      setDeleteConfirm({ isOpen: false, tenantId: '', tenantName: '' });
    }
  }

  // Upload Logo
  async function handleUploadLogo(e: React.FormEvent) {
    e.preventDefault();
    if (!uploadingTenant || !selectedFile) return;

    if (selectedFile.size > 512 * 1024) {
      setUploadError('Ukuran file melebihi batas 512 KB.');
      return;
    }

    const data = new FormData();
    data.append('logo', selectedFile);

    try {
      const res = await fetch(`/api/v1/admin/tenants/${uploadingTenant.id}/logo`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${adminToken}` },
        body: data,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        setUploadError(err.error || 'Upload gagal.');
        return;
      }

      closeUploadModal();
      showToast('Logo berhasil diunggah.', 'success');
      fetchAdminTenants(adminToken);
      onTenantsChanged?.();
    } catch {
      setUploadError('Terjadi kesalahan koneksi.');
    }
  }

  // Export JSON
  function handleExport() {
    window.open(`/api/v1/admin/export?token=${encodeURIComponent(adminToken)}`, '_blank');
  }

  // Import JSON
  async function handleImport(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      const content = await file.text();
      const res = await fetch('/api/v1/admin/import', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adminToken}`,
        },
        body: content,
      });
      if (res.ok) {
        const result = await res.json();
        showToast(`Berhasil mengimpor ${result.imported} tenant.`, 'success');
        fetchAdminTenants(adminToken);
        onTenantsChanged?.();
      } else {
        const err = await res.json().catch(() => ({}));
        showToast(err.error || 'Gagal mengimpor file.', 'error');
      }
    } catch {
      showToast('File JSON tidak valid.', 'error');
    } finally {
      e.target.value = '';
    }
  }

  // Copy Agent Setup Prompt
  function handleCopyAgentPrompt(t: Tenant) {
    const siteKey = (t as unknown as Record<string, string>)['site_key'] || '';
    const origin = window.location.origin;
    const promptText = `Tolong pasang script telemetri DioramaOps ke project website '${t.name}':
- URL Script: ${origin}/m.js
- Site Key: ${siteKey}

Panduan integrasi sesuai framework yang digunakan:
1. Next.js (App Router / Pages Router):
   Gunakan komponen resmi 'next/script' dengan strategy="afterInteractive" di root layout (misal app/layout.tsx atau pages/_app.tsx) untuk mencegah React Hydration Error (#418):
   import Script from 'next/script';
   <Script src="${origin}/m.js" data-key="${siteKey}" strategy="afterInteractive" />

2. Nuxt / Vue 3:
   Tambahkan di app.vue atau nuxt.config.ts menggunakan useHead:
   useHead({
     script: [{ src: '${origin}/m.js', 'data-key': '${siteKey}', defer: true }]
   })

3. HTML Polos / PHP / Laravel Blade / Astro / Vite SPA (React / Vue / Svelte):
   Sisipkan tag script berikut di dalam <head> atau sebelum penutup </body>:
   <script defer src="${origin}/m.js" data-key="${siteKey}"></script>

Catatan: Pastikan domain website sudah terdaftar di allowlist origin DioramaOps.`;

    navigator.clipboard.writeText(promptText);
    setPromptCopiedId(t.id);
    setTimeout(() => setPromptCopiedId(null), 2500);
  }

  // Copy Site Key
  function handleCopySiteKey(t: Tenant) {
    const siteKey = (t as unknown as Record<string, string>)['site_key'] || '';
    navigator.clipboard.writeText(siteKey);
    setKeyCopiedId(t.id);
    setTimeout(() => setKeyCopiedId(null), 2500);
  }

  // Login Gate
  if (!adminToken) {
    return (
      <div className="w-full h-full flex items-center justify-center p-4 bg-background text-foreground">
        <div className="w-full max-w-sm p-6 rounded-lg bg-surface border border-border shadow-xl">
          <div className="flex items-center space-x-2.5 mb-4">
            <div className="w-8 h-8 rounded bg-surface-subtle border border-accent/40 flex items-center justify-center text-accent">
              <Layers className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-foreground">Projects Portal</h2>
              <p className="text-xs text-foreground-muted">Otentikasi Rahasia Diperlukan</p>
            </div>
          </div>

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label htmlFor={secretInputId} className="block text-xs text-foreground-muted mb-1.5">
                ADMIN_SECRET
              </label>
              <input
                id={secretInputId}
                type="password"
                value={tokenInput}
                onChange={(e) => setTokenInput(e.target.value)}
                placeholder="Masukkan kode rahasia..."
                className="w-full px-3 py-2 rounded bg-surface-subtle border border-border text-sm text-foreground focus:outline-none focus:border-accent"
                autoFocus
              />
            </div>

            {errorMsg && <p className="text-xs text-rose-400">{errorMsg}</p>}

            <div className="flex items-center justify-between pt-2">
              <button
                type="button"
                onClick={onBackToScene}
                className="text-xs text-foreground-dim hover:text-foreground flex items-center space-x-1"
              >
                <ChevronLeft className="w-3.5 h-3.5" />
                <span>Kembali</span>
              </button>
              <button
                type="submit"
                className="px-4 py-2 rounded bg-accent hover:bg-accent-hover text-white text-xs font-medium transition-colors"
              >
                Masuk
              </button>
            </div>
          </form>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-full flex flex-col bg-background text-foreground overflow-hidden font-sans">
      {/* Header */}
      <header className="px-6 py-3.5 bg-surface border-b border-border flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <button
            onClick={onBackToScene}
            className="p-1.5 rounded hover:bg-surface-subtle text-foreground-dim hover:text-foreground transition-colors mr-1"
            title="Kembali ke Diorama 3D"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <div className="w-7 h-7 rounded bg-accent/20 border border-accent/40 flex items-center justify-center text-accent">
            <Layers className="w-4 h-4" />
          </div>
          <div>
            <h1 className="text-sm font-semibold text-foreground">Projects & Telemetry</h1>
            <p className="text-[11px] text-foreground-muted">Kelola project, origin allowlist, dan setup telemetri</p>
          </div>
        </div>

        <div className="flex items-center space-x-2.5">
          <button
            onClick={handleExport}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded bg-surface-subtle hover:bg-surface-muted border border-border text-xs text-foreground transition-colors"
            title="Download tenants.config.json"
          >
            <Download className="w-3.5 h-3.5 text-foreground-muted" />
            <span className="hidden sm:inline">Export JSON</span>
          </button>

          <label htmlFor={fileInputId} className="flex items-center space-x-1.5 px-3 py-1.5 rounded bg-surface-subtle hover:bg-surface-muted border border-border text-xs text-foreground cursor-pointer transition-colors">
            <Upload className="w-3.5 h-3.5 text-foreground-muted" />
            <span className="hidden sm:inline">Import JSON</span>
            <input
              id={fileInputId}
              type="file"
              accept=".json"
              className="hidden"
              onChange={handleImport}
            />
          </label>

          <button
            onClick={handleOpenCreate}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors bg-accent hover:bg-accent-hover text-white"
            title="Tambah Tenant"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Tambah Tenant</span>
          </button>

          <button
            onClick={handleLogout}
            className="px-2.5 py-1.5 rounded hover:bg-surface-subtle text-xs text-rose-400 transition-colors ml-2"
          >
            Keluar
          </button>
        </div>
      </header>

      {/* Main Body Table */}
      <main className="flex-1 overflow-y-auto p-6 max-w-7xl mx-auto w-full">

        {loading ? (
          <div className="h-64 flex items-center justify-center text-xs text-foreground-muted">
            Memuat daftar tenant...
          </div>
        ) : (tenants?.length ?? 0) === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center text-center p-6 border border-dashed border-border rounded-lg">
            <Globe className="w-8 h-8 text-foreground-dim mb-2" />
            <p className="text-sm font-medium text-foreground">Belum ada tenant</p>
            <p className="text-xs text-foreground-muted mt-1 mb-4">
              Daftarkan website pertama Anda untuk ditampilkan di diorama 3D.
            </p>
            <div className="flex items-center space-x-3">
              <button
                onClick={handleOpenCreate}
                className="px-3 py-1.5 rounded bg-accent hover:bg-accent-hover text-white text-xs font-medium flex items-center space-x-1.5 transition-colors"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Tambah Tenant Manual</span>
              </button>
              <label
                htmlFor={fileInputId}
                className="px-3 py-1.5 rounded border border-border bg-surface-subtle hover:bg-surface-muted text-foreground text-xs font-medium cursor-pointer flex items-center space-x-1.5 transition-colors"
              >
                <Upload className="w-3.5 h-3.5 text-foreground-dim" />
                <span>Import dari JSON</span>
              </label>
            </div>
          </div>
        ) : (
          <div className="border border-border rounded-lg overflow-hidden bg-surface">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-border bg-surface-subtle text-foreground-muted">
                  <th className="py-3 px-4 font-medium">Tenant</th>
                  <th className="py-3 px-4 font-medium">Site Key</th>
                  <th className="py-3 px-4 font-medium">Aksen</th>
                  <th className="py-3 px-4 font-medium">Status & Live</th>
                  <th className="py-3 px-4 font-medium text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {tenants.map((t) => {
                  const siteKey = (t as unknown as Record<string, string>)['site_key'] || '';

                  return (
                    <tr key={t.id} className="hover:bg-surface-subtle/50 transition-colors">
                      <td className="py-3 px-4">
                        <div className="flex items-center space-x-3">
                          <TenantAvatar
                            id={t.id}
                            name={t.name}
                            accent={t.accent}
                            logoPath={t.logo_path}
                            useFavicon={t.use_favicon}
                            size="sm"
                          />
                          <div>
                            <div className="font-semibold text-foreground flex items-center space-x-1.5">
                              <span>{t.name}</span>
                              <span className="text-[10px] text-foreground-dim font-mono">({t.id})</span>
                            </div>
                            <a
                              href={t.url}
                              target="_blank"
                              rel="noreferrer"
                              className="text-[11px] text-foreground-muted hover:text-accent flex items-center space-x-1 truncate max-w-[200px]"
                            >
                              <span>{t.url.replace(/^https?:\/\//, '')}</span>
                              <ExternalLink className="w-2.5 h-2.5" />
                            </a>
                          </div>
                        </div>
                      </td>

                      <td className="py-3 px-4">
                        <div className="flex items-center space-x-2">
                          <code className="px-2 py-0.5 rounded bg-surface-subtle border border-border text-[11px] text-foreground font-mono">
                            {siteKey ? `${siteKey.substring(0, 10)}...` : 'n/a'}
                          </code>
                          <button
                            onClick={() => handleCopySiteKey(t)}
                            className="p-1 rounded hover:bg-surface-subtle text-foreground-dim hover:text-foreground"
                            title="Salin Site Key"
                          >
                            {keyCopiedId === t.id ? (
                              <Check className="w-3.5 h-3.5 text-emerald-400" />
                            ) : (
                              <Copy className="w-3.5 h-3.5" />
                            )}
                          </button>
                        </div>
                      </td>

                      <td className="py-3 px-4">
                        <div className="flex items-center space-x-2">
                          <span
                            className="w-3 h-3 rounded-full border border-white/20"
                            style={{ backgroundColor: t.accent }}
                          />
                          <span className="font-mono text-[11px] text-foreground-muted">{t.accent}</span>
                        </div>
                      </td>

                      <td className="py-3 px-4">
                        <div className="flex items-center space-x-2">
                          <span
                            className={`w-2 h-2 rounded-full ${
                              t.status === 'up'
                                ? 'bg-emerald-500'
                                : t.status === 'degraded'
                                ? 'bg-amber-500'
                                : 'bg-rose-500'
                            }`}
                          />
                          <span className="capitalize text-foreground-muted">{t.status}</span>
                          <span className="text-foreground-dim">·</span>
                          <span className="font-semibold text-foreground">{t.active} live</span>
                        </div>
                      </td>

                      <td className="py-3 px-4 text-right">
                        <div className="flex items-center justify-end space-x-1.5">
                          {/* Copy Agent Setup Prompt */}
                          <button
                            onClick={() => handleCopyAgentPrompt(t)}
                            className="flex items-center space-x-1 px-2 py-1 rounded bg-surface-subtle hover:bg-surface-muted border border-border text-[11px] text-accent transition-colors"
                            title="Salin Agent Setup Prompt untuk AI coding agent"
                          >
                            {promptCopiedId === t.id ? (
                              <>
                                <Check className="w-3 h-3 text-emerald-400" />
                                <span className="text-emerald-400">Tersalin!</span>
                              </>
                            ) : (
                              <>
                                <Bot className="w-3 h-3" />
                                <span>Copy Agent Prompt</span>
                              </>
                            )}
                          </button>

                          {/* Upload Logo */}
                          <button
                            onClick={() => {
                              setUploadingTenant(t);
                              setSelectedFile(null);
                              setUploadError('');
                              setUploadModalOpen(true);
                            }}
                            className="p-1.5 rounded hover:bg-surface-subtle text-foreground-dim hover:text-foreground"
                            title="Upload Logo (PNG/WebP/SVG)"
                          >
                            <Upload className="w-3.5 h-3.5" />
                          </button>

                          {/* Edit */}
                          <button
                            onClick={() => handleOpenEdit(t)}
                            className="p-1.5 rounded hover:bg-surface-subtle text-foreground-dim hover:text-foreground"
                            title="Edit Tenant"
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>

                          {/* Delete */}
                          <button
                            onClick={() => handleDelete(t.id)}
                            className="p-1.5 rounded hover:bg-surface-subtle text-foreground-dim hover:text-rose-400"
                            title="Hapus Tenant"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </main>

      {/* Modal: Create/Edit Tenant */}
      {editModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-[2px]"
          onClick={() => setEditModalOpen(false)}
        >
          <div
            className="w-full max-w-lg p-6 rounded-lg bg-[#0c0c0f] border border-zinc-800 text-foreground overflow-y-auto max-h-[90vh]"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-foreground">
                {isEditing ? `Edit Tenant: ${formData.id}` : 'Tambah Tenant Baru'}
              </h3>
              <button
                onClick={() => setEditModalOpen(false)}
                className="p-1 rounded text-foreground-dim hover:text-foreground"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleSaveTenant} className="space-y-3.5 text-xs">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-foreground-muted mb-1">ID (Slug) *</label>
                  <input
                    type="text"
                    required
                    disabled={isEditing}
                    value={formData.id}
                    onChange={(e) => setFormData({ ...formData, id: e.target.value })}
                    placeholder="misal: my-app"
                    className="w-full px-3 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono disabled:opacity-50"
                  />
                </div>
                <div>
                  <label className="block text-foreground-muted mb-1">Nama Tampilan *</label>
                  <input
                    type="text"
                    required
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="misal: My Cloud App"
                    className="w-full px-3 py-1.5 rounded bg-surface-subtle border border-border text-foreground"
                  />
                </div>
              </div>

              <div>
                <label className="block text-foreground-muted mb-1">Website URL *</label>
                <input
                  type="url"
                  required
                  value={formData.url}
                  onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                  placeholder="https://app.domain.com"
                  className="w-full px-3 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-foreground-muted mb-1">Aksen Neon (Hex)</label>
                  <div className="flex items-center space-x-2">
                    <input
                      type="color"
                      value={formData.accent}
                      onChange={(e) => setFormData({ ...formData, accent: e.target.value })}
                      className="w-7 h-7 rounded border border-border bg-transparent cursor-pointer"
                    />
                    <input
                      type="text"
                      value={formData.accent}
                      onChange={(e) => setFormData({ ...formData, accent: e.target.value })}
                      className="w-full px-2 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono"
                    />
                  </div>
                  {/* Neon Color Presets */}
                  <div className="flex items-center gap-1.5 mt-2">
                    {NEON_PRESETS.map((p) => (
                      <button
                        key={p.hex}
                        type="button"
                        onClick={() => setFormData({ ...formData, accent: p.hex })}
                        title={p.label}
                        className={`w-4 h-4 rounded-full border transition-transform hover:scale-110 ${
                          formData.accent.toLowerCase() === p.hex.toLowerCase()
                            ? 'ring-2 ring-white/60 ring-offset-1 ring-offset-zinc-900 border-white'
                            : 'border-zinc-700/60'
                        }`}
                        style={{ backgroundColor: p.hex }}
                      />
                    ))}
                  </div>
                </div>

                <div className="flex items-center pt-5">
                  <label className="flex items-center space-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={formData.use_favicon}
                      onChange={(e) => setFormData({ ...formData, use_favicon: e.target.checked })}
                      className="rounded border-border text-accent focus:ring-0"
                    />
                    <span className="text-foreground-muted">Ambil Favicon Otomatis</span>
                  </label>
                </div>
              </div>

              <div>
                <label className="block text-foreground-muted mb-1">
                  Origins Allowlist (Pisahkan koma)
                </label>
                <input
                  type="text"
                  value={formData.origins}
                  onChange={(e) => setFormData({ ...formData, origins: e.target.value })}
                  placeholder="https://app.domain.com, http://localhost:3000"
                  className="w-full px-3 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono"
                />
              </div>

              <div className="grid grid-cols-3 gap-2">
                <div>
                  <label className="block text-foreground-muted mb-1">Posisi X (Opt)</label>
                  <input
                    type="number"
                    step="any"
                    value={formData.position_x}
                    onChange={(e) => setFormData({ ...formData, position_x: e.target.value })}
                    placeholder="Auto"
                    className="w-full px-2 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono"
                  />
                </div>
                <div>
                  <label className="block text-foreground-muted mb-1">Posisi Y (Opt)</label>
                  <input
                    type="number"
                    step="any"
                    value={formData.position_y}
                    onChange={(e) => setFormData({ ...formData, position_y: e.target.value })}
                    placeholder="0"
                    className="w-full px-2 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono"
                  />
                </div>
                <div>
                  <label className="block text-foreground-muted mb-1">Posisi Z (Opt)</label>
                  <input
                    type="number"
                    step="any"
                    value={formData.position_z}
                    onChange={(e) => setFormData({ ...formData, position_z: e.target.value })}
                    placeholder="Auto"
                    className="w-full px-2 py-1.5 rounded bg-surface-subtle border border-border text-foreground font-mono"
                  />
                </div>
              </div>

              <div className="pt-3 border-t border-border flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => setEditModalOpen(false)}
                  className="px-3 py-1.5 rounded border border-border hover:bg-surface-subtle text-foreground-dim"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  className="px-4 py-1.5 rounded bg-accent hover:bg-accent-hover text-white font-medium"
                >
                  Simpan Tenant
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal: Upload Logo */}
      {uploadModalOpen && uploadingTenant && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
          onClick={closeUploadModal}
        >
          <div
            className="w-full max-w-md p-6 rounded-xl bg-[#0e1017] border border-zinc-800 text-foreground shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-foreground">
                Upload Logo: {uploadingTenant.name}
              </h3>
              <button
                onClick={closeUploadModal}
                className="p-1 rounded text-foreground-dim hover:text-foreground"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleUploadLogo} className="space-y-4 text-xs">
              <div>
                <p className="text-foreground-muted mb-3 text-[11px]">
                  Format: PNG, WebP, SVG (disanitasi otomatis). Maksimal 512 KB.
                </p>

                {/* Drag and Drop Zone */}
                <div
                  onDragOver={(e) => {
                    e.preventDefault();
                    setIsDragging(true);
                  }}
                  onDragLeave={() => setIsDragging(false)}
                  onDrop={(e) => {
                    e.preventDefault();
                    setIsDragging(false);
                    const file = e.dataTransfer.files?.[0];
                    if (file) {
                      setSelectedFile(file);
                      setUploadError('');
                      if (file.type.startsWith('image/')) {
                        setFilePreview(URL.createObjectURL(file));
                      } else {
                        setFilePreview(null);
                      }
                    }
                  }}
                  onClick={() => fileInputRef.current?.click()}
                  className={`relative border-2 border-dashed rounded-xl p-5 text-center transition-all cursor-pointer ${
                    isDragging
                      ? 'border-accent bg-accent/10'
                      : 'border-zinc-700/80 bg-zinc-900/50 hover:border-zinc-500 hover:bg-zinc-900/80'
                  }`}
                >
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".png,.webp,.svg"
                    className="hidden"
                    onChange={(e) => {
                      const file = e.target.files?.[0] || null;
                      setSelectedFile(file);
                      setUploadError('');
                      if (file && file.type.startsWith('image/')) {
                        setFilePreview(URL.createObjectURL(file));
                      } else {
                        setFilePreview(null);
                      }
                    }}
                  />

                  {selectedFile ? (
                    <div className="flex flex-col items-center space-y-2">
                      {filePreview ? (
                        <img
                          src={filePreview}
                          alt="Preview"
                          className="w-12 h-12 object-contain rounded-lg border border-zinc-800 bg-zinc-950 p-1"
                        />
                      ) : (
                        <UploadCloud className="w-8 h-8 text-accent mx-auto" />
                      )}
                      <div className="text-xs font-medium text-foreground">{selectedFile.name}</div>
                      <div className="text-[10px] text-foreground-dim">
                        {(selectedFile.size / 1024).toFixed(1)} KB
                      </div>
                      <div className="text-[10px] text-accent hover:underline">
                        Klik atau seret file lain untuk mengganti
                      </div>
                    </div>
                  ) : (
                    <div className="flex flex-col items-center space-y-2">
                      <UploadCloud className="w-8 h-8 text-foreground-dim mx-auto" />
                      <div className="text-xs font-medium text-foreground">
                        Klik atau seret file logo ke sini
                      </div>
                      <div className="text-[10px] text-foreground-dim">
                        PNG, WebP, SVG (Maks. 512 KB)
                      </div>
                    </div>
                  )}
                </div>
              </div>

              {uploadError && <p className="text-xs text-rose-400">{uploadError}</p>}

              <div className="pt-2 flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={closeUploadModal}
                  className="px-3 py-1.5 rounded border border-border hover:bg-surface-subtle text-foreground-dim"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={!selectedFile}
                  className="px-4 py-1.5 rounded bg-accent hover:bg-accent-hover text-white font-medium disabled:opacity-50"
                >
                  Upload & Pasang
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Toast Notifications */}
      <ToastContainer toasts={toasts} onDismiss={removeToast} />

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={deleteConfirm.isOpen}
        title="Konfirmasi Hapus Tenant"
        message={`Apakah Anda yakin ingin menghapus tenant '${deleteConfirm.tenantName || deleteConfirm.tenantId}' secara permanen? Data telemetry yang terkait akan dihapus.`}
        confirmLabel="Hapus Permanen"
        cancelLabel="Batal"
        isDestructive={true}
        onConfirm={() => executeDelete(deleteConfirm.tenantId)}
        onCancel={() => setDeleteConfirm({ isOpen: false, tenantId: '', tenantName: '' })}
      />
    </div>
  );
}
