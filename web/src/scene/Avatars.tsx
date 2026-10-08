import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { AvatarItem, Tenant } from '../types';

interface AvatarsProps {
  avatars: AvatarItem[];
  tenants?: Tenant[];
}

function AgentAvatar({
  avatar,
  index,
  accentColor,
}: {
  avatar: AvatarItem;
  index: number;
  accentColor: string;
}) {
  const groupRef = useRef<THREE.Group>(null);

  useFrame((state) => {
    if (!groupRef.current) return;
    const now = performance.now();
    const elapsedClock = state.clock.getElapsedTime();

    const walkDuration = 2400; // 2.4 seconds to walk from lobby to booth
    const elapsed = now - avatar.spawnTime;
    const t = Math.min(1, Math.max(0, elapsed / walkDuration));

    // Ease-out cubic for smooth deceleration near booth
    const easeT = 1 - Math.pow(1 - t, 3);

    const startX = 0;
    const startZ = 0;
    const targetX = avatar.targetPos[0];
    const targetZ = avatar.targetPos[2];

    const currentX = THREE.MathUtils.lerp(startX, targetX, easeT);
    const currentZ = THREE.MathUtils.lerp(startZ, targetZ, easeT);

    let currentY = 0;
    let scale = 1.0;

    if (avatar.state === 'walking' && t < 1) {
      // Bobbing stride while walking
      currentY = Math.abs(Math.sin(elapsedClock * 14)) * 0.08;
    } else if (avatar.state === 'leaving') {
      // Despawn shrink/fade
      scale = Math.max(0.01, 1 - Math.sin((elapsedClock * 4) % Math.PI));
      currentY = (1 - scale) * 0.3;
    } else {
      // Gentle breathing bob while idle at booth
      currentY = Math.sin(elapsedClock * 2.5 + index) * 0.02;
    }

    // Calculate facing direction
    let rotY = 0;
    if (t < 0.99) {
      rotY = Math.atan2(targetX - startX, targetZ - startZ);
    } else {
      rotY = Math.atan2(-targetX, -targetZ);
    }

    groupRef.current.position.set(currentX, currentY, currentZ);
    groupRef.current.rotation.set(0, rotY, 0);
    groupRef.current.scale.set(scale, scale, scale);
  });

  return (
    <group ref={groupRef}>
      {/* Cute Low-poly Agent Body matching prototype */}
      <mesh position={[0, 0.3, 0]} castShadow>
        <cylinderGeometry args={[0.12, 0.18, 0.4, 6]} />
        <meshStandardMaterial color="#e0e0e0" roughness={0.5} />
      </mesh>

      {/* Cute Low-poly Agent Head matching prototype */}
      <mesh position={[0, 0.58, 0]} castShadow>
        <sphereGeometry args={[0.12, 8, 8]} />
        <meshStandardMaterial
          color={accentColor}
          emissive={accentColor}
          emissiveIntensity={0.3}
          roughness={0.2}
        />
      </mesh>
    </group>
  );
}

export function Avatars({ avatars, tenants = [] }: AvatarsProps) {
  const tenantColorMap = useMemo(() => {
    const map = new Map<string, string>();
    tenants.forEach((t) => {
      map.set(t.id, t.accent || '#3186ff');
    });
    return map;
  }, [tenants]);

  const visibleAvatars = useMemo(() => avatars.slice(0, 48), [avatars]);

  return (
    <group>
      {visibleAvatars.map((avatar, idx) => {
        const color = tenantColorMap.get(avatar.tenantId) || '#3186ff';
        return (
          <AgentAvatar
            key={avatar.vid}
            avatar={avatar}
            index={idx}
            accentColor={color}
          />
        );
      })}
    </group>
  );
}
