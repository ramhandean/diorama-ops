import React, { useEffect } from 'react';
import { AlertTriangle, X } from 'lucide-react';

interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  isDestructive?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmModal: React.FC<ConfirmModalProps> = ({
  isOpen,
  title,
  message,
  confirmLabel = 'Konfirmasi',
  cancelLabel = 'Batal',
  isDestructive = true,
  onConfirm,
  onCancel,
}) => {
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onCancel]);

  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-150"
      onClick={onCancel}
    >
      <div
        className="relative w-full max-w-sm rounded-xl bg-[#0e1017] border border-zinc-800 p-5 text-zinc-100 shadow-xl space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between">
          <div className="flex items-center space-x-2.5">
            <div className="w-8 h-8 rounded-lg bg-rose-950/60 border border-rose-800/50 flex items-center justify-center text-rose-400 shrink-0">
              <AlertTriangle className="w-4 h-4" />
            </div>
            <h3 id="confirm-modal-title" className="text-sm font-semibold text-white">
              {title}
            </h3>
          </div>
          <button
            onClick={onCancel}
            aria-label="Tutup dialog"
            className="p-1 text-zinc-500 hover:text-zinc-200 rounded-lg hover:bg-zinc-800 transition-colors focus:outline-none focus:ring-1 focus:ring-zinc-400"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <p className="text-xs text-zinc-300 leading-relaxed">
          {message}
        </p>

        <div className="pt-2 flex items-center justify-end space-x-2">
          <button
            type="button"
            onClick={onCancel}
            className="min-h-[44px] px-3.5 py-2 rounded-lg border border-zinc-700/80 bg-zinc-900 hover:bg-zinc-800 text-xs font-medium text-zinc-300 transition-colors focus:outline-none focus:ring-1 focus:ring-zinc-400"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={`min-h-[44px] px-4 py-2 rounded-lg text-xs font-medium transition-colors focus:outline-none focus:ring-2 ${
              isDestructive
                ? 'bg-rose-600 hover:bg-rose-500 active:bg-rose-700 text-white focus:ring-rose-400 shadow-md shadow-rose-950/30'
                : 'bg-blue-600 hover:bg-blue-500 text-white focus:ring-blue-400'
            }`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
};
