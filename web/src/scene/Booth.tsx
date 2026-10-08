import { useState, useMemo, useRef, useEffect } from 'react';
import { useFrame } from '@react-three/fiber';
import { Html } from '@react-three/drei';
import * as THREE from 'three';
import { Tenant } from '../types';

interface BoothProps {
  tenant: Tenant;
  position: [number, number, number];
  rotationY: number;
  totalTenants: number;
  onSelect: (t: Tenant) => void;
}

export function Booth({
  tenant,
  position,
  rotationY,
  onSelect,
}: BoothProps) {
  const [hovered, setHovered] = useState(false);
  const [logoTexture, setLogoTexture] = useState<THREE.Texture | null>(null);
  const holoRef = useRef<THREE.Group>(null);
  const roofEdges = useMemo(
    () => new THREE.EdgesGeometry(new THREE.BoxGeometry(1.7, 0.04, 1.5)),
    []
  );

  // Status and accent color
  const statusColor = useMemo(() => {
    if (tenant.status === 'down') return '#ef4444';
    if (tenant.status === 'degraded') return '#f59e0b';
    return tenant.accent || '#3186ff';
  }, [tenant.status, tenant.accent]);

  // Determine hologram shape deterministically (fallback when no logo)
  const iconShape = useMemo(() => {
    const shapes = ['box', 'cylinder', 'octahedron', 'torus'] as const;
    let hash = 0;
    for (let i = 0; i < tenant.id.length; i++) {
      hash = (hash << 5) - hash + tenant.id.charCodeAt(i);
    }
    return shapes[Math.abs(hash) % shapes.length];
  }, [tenant.id]);

  const logoUrl = useMemo(
    () => tenant.logo_path || (tenant.use_favicon ? `/api/v1/favicon/${tenant.id}` : null),
    [tenant.logo_path, tenant.use_favicon, tenant.id]
  );

  // Load logo texture for 3D hologram display with auto-trimming padding & GPU memory disposal
  useEffect(() => {
    if (!logoUrl) {
      setLogoTexture(null);
      return;
    }
    let isCancelled = false;
    let currentTex: THREE.Texture | null = null;
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      if (isCancelled) return;
      const targetSize = 256;
      const canvas = document.createElement('canvas');
      canvas.width = targetSize;
      canvas.height = targetSize;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      const w = img.naturalWidth || img.width || 32;
      const h = img.naturalHeight || img.height || 32;
      const measureCanvas = document.createElement('canvas');
      measureCanvas.width = w;
      measureCanvas.height = h;
      const mCtx = measureCanvas.getContext('2d');

      if (mCtx) {
        mCtx.drawImage(img, 0, 0);
        try {
          const imgData = mCtx.getImageData(0, 0, w, h);
          let minX = w, minY = h, maxX = 0, maxY = 0;
          let hasContent = false;
          let transparentCount = 0;
          let opaqueCount = 0;
          let totalLum = 0;

          for (let y = 0; y < h; y++) {
            for (let x = 0; x < w; x++) {
              const idx = (y * w + x) * 4;
              const r = imgData.data[idx];
              const g = imgData.data[idx + 1];
              const b = imgData.data[idx + 2];
              const alpha = imgData.data[idx + 3];
              if (alpha > 25) {
                hasContent = true;
                opaqueCount++;
                totalLum += 0.299 * r + 0.587 * g + 0.114 * b;
                if (x < minX) minX = x;
                if (x > maxX) maxX = x;
                if (y < minY) minY = y;
                if (y > maxY) maxY = y;
              } else {
                transparentCount++;
              }
            }
          }

          const isMostlyTransparent = transparentCount > (w * h) * 0.15;
          const avgLum = opaqueCount > 0 ? totalLum / opaqueCount : 128;
          const isDarkLogo = avgLum < 110;

          if (isMostlyTransparent && isDarkLogo) {
            ctx.fillStyle = '#ffffff';
            ctx.beginPath();
            ctx.arc(targetSize / 2, targetSize / 2, targetSize / 2, 0, Math.PI * 2);
            ctx.fill();
          }

          if (hasContent && minX < maxX && minY < maxY) {
            const cropW = maxX - minX + 1;
            const cropH = maxY - minY + 1;
            const maxDim = Math.max(cropW, cropH);
            const padRatio = 0.90;
            const drawSize = targetSize * padRatio;
            const scale = drawSize / maxDim;
            const finalW = cropW * scale;
            const finalH = cropH * scale;
            const dx = (targetSize - finalW) / 2;
            const dy = (targetSize - finalH) / 2;

            ctx.drawImage(img, minX, minY, cropW, cropH, dx, dy, finalW, finalH);
          } else {
            ctx.drawImage(img, 0, 0, targetSize, targetSize);
          }
        } catch {
          ctx.drawImage(img, 0, 0, targetSize, targetSize);
        }
      } else {
        ctx.drawImage(img, 0, 0, targetSize, targetSize);
      }

      const tex = new THREE.CanvasTexture(canvas);
      tex.colorSpace = THREE.SRGBColorSpace;
      tex.minFilter = THREE.LinearFilter;
      tex.generateMipmaps = false;
      currentTex = tex;
      setLogoTexture(tex);
    };
    img.onerror = () => {
      if (!isCancelled) setLogoTexture(null);
    };
    img.src = logoUrl;

    return () => {
      isCancelled = true;
      currentTex?.dispose();
    };
  }, [logoUrl]);

  // Hologram rotation & floating bobbing animation
  useFrame((_, delta) => {
    if (!holoRef.current) return;
    const time = performance.now() * 0.001;
    holoRef.current.rotation.y += delta * 0.8;
    holoRef.current.rotation.x = Math.sin(time * 1.5) * 0.1;
    holoRef.current.position.y = 1.6 + Math.sin(time * 2) * 0.08;
  });

  return (
    <group
      position={position}
      rotation={[0, rotationY, 0]}
      onClick={(e) => {
        e.stopPropagation();
        onSelect(tenant);
      }}
      onPointerOver={(e) => {
        e.stopPropagation();
        setHovered(true);
      }}
      onPointerOut={() => setHovered(false)}
    >
      {/* Octagonal Base Platform */}
      <mesh position={[0, 0.1, 0]} receiveShadow castShadow>
        <cylinderGeometry args={[1.5, 1.6, 0.2, 8]} />
        <meshStandardMaterial
          color="#1e2230"
          roughness={0.4}
          metalness={0.4}
        />
      </mesh>

      {/* Flat Neon Ring */}
      <mesh position={[0, 0.21, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[1.45, 0.03, 8, 24]} />
        <meshBasicMaterial color={hovered ? '#ffffff' : statusColor} />
      </mesh>

      {/* Canopy Roof (Translucent Tinted Cyber Glass) */}
      <mesh position={[0, 2.05, 0]}>
        <boxGeometry args={[1.7, 0.04, 1.5]} />
        <meshStandardMaterial
          color="#182030"
          roughness={0.1}
          metalness={0.8}
          transparent
          opacity={0.28}
          depthWrite={false}
        />
      </mesh>

      {/* Canopy Roof Neon Edge Outline */}
      <lineSegments position={[0, 2.05, 0]} geometry={roofEdges}>
        <lineBasicMaterial color={hovered ? '#ffffff' : statusColor} />
      </lineSegments>

      {/* 2 Rear Support Pillars */}
      <mesh position={[-0.75, 1.05, -0.65]} castShadow>
        <boxGeometry args={[0.08, 1.9, 0.08]} />
        <meshStandardMaterial color="#353e56" metalness={0.8} roughness={0.2} />
      </mesh>
      <mesh position={[0.75, 1.05, -0.65]} castShadow>
        <boxGeometry args={[0.08, 1.9, 0.08]} />
        <meshStandardMaterial color="#353e56" metalness={0.8} roughness={0.2} />
      </mesh>

      {/* Low-poly Terminal Desk */}
      <mesh position={[0, 0.45, -0.35]} castShadow receiveShadow>
        <boxGeometry args={[1.25, 0.5, 0.55]} />
        <meshStandardMaterial color="#252a3c" roughness={0.3} metalness={0.6} />
      </mesh>

      {/* Screen / Neon Stripe on Desk */}
      <mesh position={[0, 0.72, -0.06]}>
        <boxGeometry args={[0.8, 0.15, 0.04]} />
        <meshBasicMaterial color={statusColor} />
      </mesh>

      {/* Light Beam Cone below Hologram */}
      <mesh position={[0, 0.78, 0]}>
        <cylinderGeometry args={[0.18, 0.6, 0.65, 24, 1, true]} />
        <meshBasicMaterial
          color={statusColor}
          transparent
          opacity={0.18}
          depthWrite={false}
          side={THREE.DoubleSide}
        />
      </mesh>

      {/* Floating 3D Hologram Logo / Geometrical Mesh */}
      <group ref={holoRef} position={[0, 1.42, 0]}>
        {logoTexture ? (
          <>
            {/* Coin Body - Rotated so circular face faces Z axis */}
            <mesh rotation={[Math.PI / 2, 0, 0]}>
              <cylinderGeometry args={[0.36, 0.36, 0.04, 32]} />
              <meshStandardMaterial
                color="#18181b"
                emissive={statusColor}
                emissiveIntensity={hovered ? 0.6 : 0.25}
                roughness={0.2}
                metalness={0.7}
              />
            </mesh>

            {/* Glowing Coin Neon Rim Frame */}
            <mesh position={[0, 0, 0]}>
              <torusGeometry args={[0.365, 0.012, 12, 32]} />
              <meshBasicMaterial color={hovered ? '#ffffff' : statusColor} />
            </mesh>

            {/* Front Contrast Backplate Disc */}
            <mesh position={[0, 0, 0.021]}>
              <circleGeometry args={[0.345, 32]} />
              <meshBasicMaterial color="#18181b" />
            </mesh>

            {/* Front Logo Face (circular full-bleed) */}
            <mesh position={[0, 0, 0.023]}>
              <circleGeometry args={[0.345, 32]} />
              <meshBasicMaterial
                map={logoTexture}
                transparent
                depthWrite={false}
              />
            </mesh>

            {/* Back Contrast Backplate Disc */}
            <mesh position={[0, 0, -0.021]} rotation={[0, Math.PI, 0]}>
              <circleGeometry args={[0.345, 32]} />
              <meshBasicMaterial color="#18181b" />
            </mesh>

            {/* Back Logo Face (mirrored so readable from reverse angle) */}
            <mesh position={[0, 0, -0.023]} rotation={[0, Math.PI, 0]}>
              <circleGeometry args={[0.345, 32]} />
              <meshBasicMaterial
                map={logoTexture}
                transparent
                depthWrite={false}
              />
            </mesh>

            {/* Angled Holographic Gyroscope Orbit Ring */}
            <mesh rotation={[Math.PI / 3, Math.PI / 4, 0]}>
              <torusGeometry args={[0.48, 0.01, 8, 32]} />
              <meshBasicMaterial
                color={statusColor}
                transparent
                opacity={hovered ? 0.8 : 0.45}
              />
            </mesh>
          </>
        ) : (
          <mesh castShadow>
            {iconShape === 'box' && <boxGeometry args={[0.6, 0.6, 0.6]} />}
            {iconShape === 'cylinder' && <cylinderGeometry args={[0.3, 0.3, 0.6, 6]} />}
            {iconShape === 'octahedron' && <octahedronGeometry args={[0.45]} />}
            {iconShape === 'torus' && <torusGeometry args={[0.35, 0.12, 8, 16]} />}
            <meshStandardMaterial
              color={statusColor}
              emissive={statusColor}
              emissiveIntensity={hovered ? 0.9 : 0.6}
              roughness={0.2}
              metalness={0.2}
              wireframe
            />
          </mesh>
        )}
      </group>

      {/* Point Light Illuminating Booth */}
      <pointLight
        position={[0, 1.45, 0]}
        color={statusColor}
        intensity={hovered ? 2.2 : 1.4}
        distance={4.0}
      />

      {/* Minimal Floating Identity Label */}
      <Html
        position={[0, 2.55, 0]}
        center
        distanceFactor={6}
        zIndexRange={[10, 0]}
        className="pointer-events-none select-none"
      >
        <div className="flex flex-col items-center gap-1">
          <div className="flex items-center space-x-2 px-3 py-1 rounded-full bg-[#0c0c0f]/90 border border-zinc-700/80 backdrop-blur-sm shadow-md whitespace-nowrap">
            <span
              className="w-2 h-2 rounded-full shrink-0 shadow-sm animate-pulse"
              style={{ backgroundColor: statusColor }}
            />
            <span className="text-[11px] font-semibold text-white tracking-wide">
              {tenant.name}
            </span>
            {tenant.active > 0 && (
              <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-emerald-950/80 text-emerald-400 font-mono border border-emerald-500/40">
                {tenant.active}
              </span>
            )}
          </div>
        </div>
      </Html>
    </group>
  );
}
