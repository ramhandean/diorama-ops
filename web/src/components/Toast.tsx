import React, { useEffect } from 'react';
import { CheckCircle2, AlertCircle, Info, AlertTriangle, X } from 'lucide-react';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface ToastMessage {
  id: string;
  type: ToastType;
  message: string;
}

interface ToastContainerProps {
  toasts: ToastMessage[];
  onDismiss: (id: string) => void;
}

export const ToastContainer: React.FC<ToastContainerProps> = ({ toasts, onDismiss }) => {
  if (toasts.length === 0) return null;

  return (
    <div
      role="region"
      aria-label="Notifikasi Sistem"
      className="fixed bottom-5 right-5 z-[100] flex flex-col space-y-2 pointer-events-none max-w-sm w-full px-4 sm:px-0"
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onDismiss={onDismiss} />
      ))}
    </div>
  );
};

interface ToastItemProps {
  toast: ToastMessage;
  onDismiss: (id: string) => void;
}

const ToastItem: React.FC<ToastItemProps> = ({ toast, onDismiss }) => {
  useEffect(() => {
    const timer = setTimeout(() => {
      onDismiss(toast.id);
    }, 4000);
    return () => clearTimeout(timer);
  }, [toast.id, onDismiss]);

  const icons = {
    success: <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />,
    error: <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />,
    info: <Info className="w-4 h-4 text-blue-400 shrink-0" />,
    warning: <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0" />,
  };

  const borders = {
    success: 'border-emerald-500/30',
    error: 'border-rose-500/30',
    info: 'border-blue-500/30',
    warning: 'border-amber-500/30',
  };

  return (
    <div
      role="alert"
      className={`pointer-events-auto flex items-start space-x-3 p-3.5 rounded-xl bg-[#0e1017]/95 border ${borders[toast.type]} backdrop-blur-md shadow-lg text-xs text-zinc-200 transition-all duration-200 animate-in fade-in slide-in-from-bottom-2`}
    >
      <div className="mt-0.5">{icons[toast.type]}</div>
      <div className="flex-1 leading-relaxed font-medium text-zinc-100 break-words">
        {toast.message}
      </div>
      <button
        onClick={() => onDismiss(toast.id)}
        aria-label="Tutup notifikasi"
        className="p-1 -mr-1 -mt-1 text-zinc-500 hover:text-zinc-200 rounded-lg hover:bg-zinc-800/60 transition-colors focus:outline-none focus:ring-1 focus:ring-zinc-400"
      >
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  );
};
