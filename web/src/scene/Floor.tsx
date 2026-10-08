import { useMemo } from 'react';
import * as THREE from 'three';
import { computePlatformDimensions, LayoutBooth } from './layout';

interface FloorProps {
  tenantsCount?: number;
  booths?: LayoutBooth[];
}

export function Floor({ tenantsCount = 0, booths = [] }: FloorProps) {
  const { floorRadius, segments } = useMemo(
    () => computePlatformDimensions(tenantsCount),
    [tenantsCount]
  );

  return (
    <group position={[0, 0, 0]}>
      {/* 1. Main Base Pedestal (Adaptive Polygon Cylinder) */}
      <mesh receiveShadow position={[0, -0.175, 0]}>
        <cylinderGeometry args={[floorRadius, floorRadius * 1.04, 0.35, segments]} />
        <meshStandardMaterial
          color="#1c2234"
          roughness={0.85}
          metalness={0.15}
        />
      </mesh>

      {/* 2. Glowing Polygon Outer Rim */}
      <mesh position={[0, 0.01, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[floorRadius * 0.98, 0.045, 8, segments]} />
        <meshBasicMaterial color="#38bdf8" />
      </mesh>

      {/* 3. Inner Elevated Platform Floor with Cyber Blue Base */}
      <mesh receiveShadow position={[0, 0.02, 0]}>
        <cylinderGeometry args={[floorRadius * 0.96, floorRadius * 0.96, 0.03, segments]} />
        <meshStandardMaterial
          color="#141828"
          roughness={0.85}
          metalness={0.15}
        />
      </mesh>

      {/* 3b. Mid-Orbit Concentric Cyber Ring */}
      <mesh position={[0, 0.025, 0]} rotation={[-Math.PI / 2, 0, 0]}>
        <ringGeometry args={[floorRadius * 0.52, floorRadius * 0.535, segments * 2]} />
        <meshBasicMaterial color="#38bdf8" transparent opacity={0.35} side={THREE.DoubleSide} />
      </mesh>

      {/* 4. Radial Glowing Connector Rails leading to each booth */}
      {booths.map((b, i) => {
        const x = b.position[0];
        const z = b.position[2];
        const color = b.tenant?.accent || '#38bdf8';

        const curve = new THREE.LineCurve3(
          new THREE.Vector3(0, 0.04, 0),
          new THREE.Vector3(x, 0.04, z)
        );
        const railGeo = new THREE.TubeGeometry(curve, 10, 0.025, 6, false);

        return (
          <mesh key={b.tenant?.id || i} geometry={railGeo}>
            <meshBasicMaterial color={color} transparent opacity={0.7} />
          </mesh>
        );
      })}

      {/* 5. Central Lobby Portal Marker */}
      <group position={[0, 0.04, 0]}>
        <mesh rotation={[-Math.PI / 2, 0, 0]}>
          <ringGeometry args={[0.8, 0.95, 32]} />
          <meshBasicMaterial color="#38bdf8" side={THREE.DoubleSide} />
        </mesh>
        <pointLight position={[0, 0.4, 0]} intensity={0.35} distance={2.5} color="#38bdf8" />
      </group>
    </group>
  );
}
