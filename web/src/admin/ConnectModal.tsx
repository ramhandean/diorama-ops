import React, { useState } from 'react';
import {
  X,
  Code2,
  Sparkles,
  Copy,
  Check,
  ExternalLink,
  ShieldCheck,
} from 'lucide-react';
import { Tenant } from '../types';
import { TenantAvatar } from '../components/TenantAvatar';

interface ConnectModalProps {
  tenant: Tenant | null;
  onClose: () => void;
  onCopyPrompt: (tenant: Tenant) => void;
  onCopySiteKey: (tenant: Tenant) => void;
  promptCopied: boolean;
  keyCopied: boolean;
}

type FrameworkTab = 'html' | 'nextjs' | 'vue';

export const ConnectModal: React.FC<ConnectModalProps> = ({
  tenant,
  onClose,
  onCopyPrompt,
  onCopySiteKey,
  promptCopied,
  keyCopied,
}) => {
  const [activeTab, setActiveTab] = useState<FrameworkTab>('html');
  const [snippetCopied, setSnippetCopied] = useState(false);

  if (!tenant) return null;

  const siteKey = (tenant as unknown as Record<string, string>)['site_key'] || '';
  const origin = typeof window !== 'undefined' ? window.location.origin : 'https://diorama-ops.engineroom.my.id';

  const snippets: Record<FrameworkTab, { label: string; code: string; desc: string }> = {
    html: {
      label: 'HTML / PHP / Laravel / Astro / Vite SPA',
      code: `<!-- Tambahkan tag script ini ke dalam <head> atau sebelum </body> -->\n<script defer src="${origin}/m.js" data-key="${siteKey}"></script>`,
      desc: 'Pelacakan murni berbasis presensi memori, tanpa cookie (< 1.5 KB gzip).',
    },
    nextjs: {
      label: 'Next.js (App / Pages Router)',
      code: `// Sisipkan di root layout (app/layout.tsx atau pages/_app.tsx)\nimport Script from 'next/script';\n\n<Script\n  src="${origin}/m.js"\n  data-key="${siteKey}"\n  strategy="afterInteractive"\n/>`,
      desc: 'Gunakan strategy="afterInteractive" untuk mencegah React Hydration Error (#418).',
    },
    vue: {
      label: 'Nuxt 3 / Vue 3 (useHead)',
      code: `// Tambahkan di app.vue atau nuxt.config.ts\nuseHead({\n  script: [\n    {\n      src: '${origin}/m.js',\n      'data-key': '${siteKey}',\n      defer: true\n    }\n  ]\n})`,
      desc: 'Inject otomatis ke SSR head dan terhidrasi dengan aman di sisi klien.',
    },
  };

  const handleCopySnippet = () => {
    navigator.clipboard.writeText(snippets[activeTab].code);
    setSnippetCopied(true);
    setTimeout(() => setSnippetCopied(false), 2000);
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="connect-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-150"
      onClick={onClose}
    >
      <div
        className="relative w-full max-w-2xl max-h-[90dvh] bg-[#0d0e14] border border-zinc-800 rounded-2xl shadow-2xl flex flex-col overflow-hidden text-zinc-100 font-sans text-left"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800/80 bg-[#09090b]">
          <div className="flex items-center space-x-3 min-w-0">
            <TenantAvatar
              id={tenant.id}
              name={tenant.name}
              accent={tenant.accent}
              logoPath={tenant.logo_path}
              useFavicon={tenant.use_favicon}
              size="md"
            />
            <div className="min-w-0">
              <div className="flex items-center space-x-2 truncate">
                <h2 id="connect-modal-title" className="text-sm sm:text-base font-semibold text-white truncate">
                  Setup & Hubungkan: {tenant.name}
                </h2>
                <span className="text-[10px] text-zinc-400 font-mono px-1.5 py-0.5 rounded bg-zinc-900 border border-zinc-800">
                  {tenant.id}
                </span>
              </div>
              <a
                href={tenant.url}
                target="_blank"
                rel="noreferrer"
                className="text-xs text-zinc-400 hover:text-blue-400 flex items-center space-x-1 truncate mt-0.5"
              >
                <span className="truncate">{tenant.url}</span>
                <ExternalLink className="w-3 h-3 shrink-0" />
              </a>
            </div>
          </div>

          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-white flex items-center justify-center transition-colors shrink-0 ml-2"
            aria-label="Tutup modal"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content Body */}
        <div className="p-5 overflow-y-auto space-y-6 text-xs text-zinc-300">
          {/* Site Key Quick Bar */}
          <div className="p-3 rounded-xl bg-[#12131a] border border-zinc-800/90 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
            <div className="min-w-0 space-y-0.5">
              <span className="text-[11px] text-zinc-400 font-medium">Public Site Key Project:</span>
              <div className="font-mono text-xs text-white select-all break-all">{siteKey || 'n/a'}</div>
            </div>
            <button
              onClick={() => onCopySiteKey(tenant)}
              className="min-h-[36px] px-3 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-700/80 text-zinc-200 text-xs font-medium flex items-center justify-center space-x-1.5 transition-colors shrink-0"
            >
              {keyCopied ? (
                <>
                  <Check className="w-3.5 h-3.5 text-emerald-400" />
                  <span className="text-emerald-400">Site Key Disalin</span>
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5 text-zinc-400" />
                  <span>Salin Site Key</span>
                </>
              )}
            </button>
          </div>

          {/* Option 1: AI Prompt Fast-Track (Recommended) */}
          <div className="p-4 rounded-xl bg-purple-950/20 border border-purple-800/40 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <div className="p-1.5 rounded-lg bg-purple-900/40 border border-purple-700/40 text-purple-300">
                  <Sparkles className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="font-semibold text-xs sm:text-sm text-white flex items-center space-x-1.5">
                    <span>Opsi 1: Otomatis via AI Coding Assistant</span>
                    <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-purple-900/80 text-purple-200 border border-purple-700/50">
                      Rekomendasi
                    </span>
                  </h3>
                  <p className="text-[11px] text-zinc-400">
                    Salin prompt dan tempelkan langsung ke Cursor, Claude Code, GitHub Copilot, atau ChatGPT.
                  </p>
                </div>
              </div>
            </div>

            <button
              onClick={() => onCopyPrompt(tenant)}
              className="w-full min-h-[44px] px-4 py-2.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white font-medium text-xs flex items-center justify-center space-x-2 transition-colors shadow-sm focus:outline-none focus:ring-2 focus:ring-purple-400"
            >
              {promptCopied ? (
                <>
                  <Check className="w-4 h-4 text-emerald-300" />
                  <span>Prompt AI Tersalin ke Clipboard!</span>
                </>
              ) : (
                <>
                  <Sparkles className="w-4 h-4" />
                  <span>Salin Prompt Integrasi AI</span>
                </>
              )}
            </button>
          </div>

          {/* Option 2: Manual Script Tag */}
          <div className="p-4 rounded-xl bg-[#12131a] border border-zinc-800/90 space-y-3">
            <div className="flex items-center space-x-2">
              <div className="p-1.5 rounded-lg bg-blue-950/40 border border-blue-800/30 text-blue-400">
                <Code2 className="w-4 h-4" />
              </div>
              <div>
                <h3 className="font-semibold text-xs sm:text-sm text-white">
                  Opsi 2: Pasang Manual (1 Baris Kode)
                </h3>
                <p className="text-[11px] text-zinc-400">Pilih framework dan sisipkan tag telemetri ke project Anda.</p>
              </div>
            </div>

            {/* Framework Tabs */}
            <div className="flex flex-wrap items-center justify-between gap-2 pt-1 border-b border-zinc-800/80 pb-2.5">
              <div className="flex items-center space-x-1.5">
                {(['html', 'nextjs', 'vue'] as FrameworkTab[]).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setActiveTab(tab)}
                    className={`min-h-[36px] px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                      activeTab === tab
                        ? 'bg-blue-600 text-white'
                        : 'text-zinc-400 hover:text-white hover:bg-zinc-800/60'
                    }`}
                  >
                    {tab === 'html' ? 'HTML / Astro' : tab === 'nextjs' ? 'Next.js' : 'Nuxt / Vue 3'}
                  </button>
                ))}
              </div>

              <button
                onClick={handleCopySnippet}
                className="min-h-[34px] px-2.5 py-1 rounded-md bg-zinc-900 hover:bg-zinc-800 border border-zinc-700/80 text-zinc-300 hover:text-white text-xs flex items-center space-x-1.5 transition-colors"
              >
                {snippetCopied ? (
                  <>
                    <Check className="w-3.5 h-3.5 text-emerald-400" />
                    <span className="text-emerald-400">Disalin</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3.5 h-3.5" />
                    <span>Salin Kode</span>
                  </>
                )}
              </button>
            </div>

            {/* Code Snippet Box */}
            <pre className="p-3.5 rounded-xl bg-zinc-950 border border-zinc-800/90 font-mono text-[11px] text-zinc-200 overflow-x-auto leading-relaxed">
              {snippets[activeTab].code}
            </pre>
            <p className="text-[11px] text-zinc-500">{snippets[activeTab].desc}</p>
          </div>

          {/* Verification & Live Test Checklist */}
          <div className="p-4 rounded-xl bg-[#12131a] border border-zinc-800/90 space-y-2.5">
            <h4 className="font-semibold text-xs text-white flex items-center space-x-2">
              <ShieldCheck className="w-4 h-4 text-emerald-400" />
              <span>Langkah Verifikasi Koneksi</span>
            </h4>
            <ul className="space-y-1.5 text-xs text-zinc-300">
              <li className="flex items-start space-x-2">
                <span className="w-4 h-4 rounded-full bg-zinc-800 text-zinc-400 text-[10px] font-bold flex items-center justify-center shrink-0 mt-0.5">
                  1
                </span>
                <span>
                  Pastikan domain origin website Anda (<code className="text-zinc-200">{tenant.url}</code>) sudah terdaftar di allowlist origin tenant.
                </span>
              </li>
              <li className="flex items-start space-x-2">
                <span className="w-4 h-4 rounded-full bg-zinc-800 text-zinc-400 text-[10px] font-bold flex items-center justify-center shrink-0 mt-0.5">
                  2
                </span>
                <span>Deploy atau jalankan website Anda di browser (localhost atau domain live).</span>
              </li>
              <li className="flex items-start space-x-2">
                <span className="w-4 h-4 rounded-full bg-zinc-800 text-zinc-400 text-[10px] font-bold flex items-center justify-center shrink-0 mt-0.5">
                  3
                </span>
                <span>Kembali ke Diorama 3D DioramaOps: avatar pengunjung akan langsung berjalan masuk ke booth Anda!</span>
              </li>
            </ul>
          </div>
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-zinc-800/80 bg-[#09090b] flex items-center justify-end">
          <button
            onClick={onClose}
            className="min-h-[40px] px-4 py-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-700/80 text-zinc-200 text-xs font-medium transition-colors"
          >
            Selesai
          </button>
        </div>
      </div>
    </div>
  );
};
