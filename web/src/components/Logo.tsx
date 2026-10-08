import React from 'react';

interface LogoProps {
  className?: string;
  size?: number;
  glow?: boolean;
}

export const Logo: React.FC<LogoProps> = ({ className = '', size = 32, glow = true }) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={`inline-block select-none ${className}`}
      aria-label="DioramaOps Logo"
    >
      <defs>
        {glow && (
          <filter id="neon-glow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
        )}
        <linearGradient id="cyber-cyan" x1="15" y1="15" x2="85" y2="85" gradientUnits="userSpaceOnUse">
          <stop stopColor="#38bdf8" />
          <stop offset="1" stopColor="#2563eb" />
        </linearGradient>
      </defs>

      {/* Isometric Cube Outer Edges */}
      <polygon
        points="50,14 82,32 82,68 50,86 18,68 18,32"
        stroke="url(#cyber-cyan)"
        strokeWidth="3.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        filter={glow ? 'url(#neon-glow)' : undefined}
      />

      {/* Internal Axis Rays */}
      <line
        x1="50"
        y1="50"
        x2="50"
        y2="14"
        stroke="#38bdf8"
        strokeWidth="2.5"
        strokeOpacity="0.8"
      />
      <line
        x1="50"
        y1="50"
        x2="82"
        y2="68"
        stroke="#3b82f6"
        strokeWidth="2.5"
        strokeOpacity="0.8"
      />
      <line
        x1="50"
        y1="50"
        x2="18"
        y2="68"
        stroke="#38bdf8"
        strokeWidth="2.5"
        strokeOpacity="0.8"
      />

      {/* Central Telemetry Hexagon Core */}
      <polygon
        points="50,44 55,47 55,53 50,56 45,53 45,47"
        fill="#38bdf8"
        stroke="#60a5fa"
        strokeWidth="1.5"
      />

      {/* Network Nodes (Top Quadrant) */}
      <line x1="50" y1="38" x2="40" y2="30" stroke="#38bdf8" strokeWidth="2" strokeLinecap="round" />
      <circle cx="40" cy="30" r="2.5" fill="#38bdf8" />

      <line x1="50" y1="34" x2="60" y2="28" stroke="#60a5fa" strokeWidth="2" strokeLinecap="round" />
      <circle cx="60" cy="28" r="2.5" fill="#60a5fa" />

      {/* Network Nodes (Left Lower Quadrant) */}
      <line x1="42" y1="54" x2="32" y2="52" stroke="#38bdf8" strokeWidth="2" strokeLinecap="round" />
      <circle cx="32" cy="52" r="2.5" fill="#38bdf8" />

      <line x1="36" y1="58" x2="30" y2="64" stroke="#60a5fa" strokeWidth="2" strokeLinecap="round" />
      <circle cx="30" cy="64" r="2.5" fill="#60a5fa" />

      {/* Network Nodes (Right Lower Quadrant) */}
      <line x1="58" y1="54" x2="68" y2="52" stroke="#38bdf8" strokeWidth="2" strokeLinecap="round" />
      <circle cx="68" cy="52" r="2.5" fill="#38bdf8" />

      <line x1="64" y1="58" x2="70" y2="64" stroke="#60a5fa" strokeWidth="2" strokeLinecap="round" />
      <circle cx="70" cy="64" r="2.5" fill="#60a5fa" />
    </svg>
  );
};
