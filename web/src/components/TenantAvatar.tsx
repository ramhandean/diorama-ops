import { useState } from 'react';

interface TenantAvatarProps {
  id?: string;
  name?: string;
  accent?: string;
  logoPath?: string;
  useFavicon?: boolean;
  size?: 'xs' | 'sm' | 'md' | 'lg';
  className?: string;
}

export function TenantAvatar({
  id,
  name = '',
  accent = '#3b82f6',
  logoPath,
  useFavicon,
  size = 'md',
  className = '',
}: TenantAvatarProps) {
  const [imgError, setImgError] = useState(false);

  const rawLogo = logoPath || (useFavicon && id ? `/api/v1/favicon/${id}` : null);
  const showImg = rawLogo && !imgError;

  const sizeClasses = {
    xs: 'w-4 h-4 text-[9px]',
    sm: 'w-8 h-8 text-xs',
    md: 'w-10 h-10 text-sm',
    lg: 'w-12 h-12 text-base',
  }[size];

  const trimmed = name.trim();
  const initials = trimmed.length > 0 ? trimmed.substring(0, 2).toUpperCase() : '??';

  if (showImg) {
    return (
      <img
        src={rawLogo}
        alt={name || 'Project Logo'}
        className={`${sizeClasses} object-contain rounded p-1 bg-white border border-border/50 shrink-0 ${className}`}
        onError={() => setImgError(true)}
      />
    );
  }

  return (
    <div
      className={`${sizeClasses} rounded flex items-center justify-center font-bold select-none shrink-0 border ${className}`}
      style={{
        backgroundColor: `${accent}22`,
        borderColor: accent,
        color: accent,
      }}
      title={name}
    >
      {initials}
    </div>
  );
}
