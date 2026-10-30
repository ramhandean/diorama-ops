import { useMemo } from 'react';
import { OrbitControls } from '@react-three/drei';
import { Tenant, AvatarItem } from '../types';
import { computeBoothLayout } from './layout';
import { Floor } from './Floor';
import { Booth } from './Booth';
import { Avatars } from './Avatars';

interface DioramaSceneProps {
  tenants: Tenant[];
  avatars: AvatarItem[];
  onSelectTenant: (t: Tenant) => void;
}

export function DioramaScene({
  tenants,
  avatars,
  onSelectTenant,
}: DioramaSceneProps) {
  const layout = useMemo(() => computeBoothLayout(tenants), [tenants]);

  return (
    <>
      {/* Subtle Fog matching prototype */}
      <fog attach="fog" args={['#0c0d14', 22, 45]} />

      {/* High-Contrast Studio Lighting Setup */}
      <ambientLight intensity={0.9} />
      <hemisphereLight args={['#dbeafe', '#1e293b', 1.1]} />
      <directionalLight
        position={[10, 16, 8]}
        intensity={1.6}
        castShadow
        shadow-mapSize={[1024, 1024]}
        shadow-camera-near={0.5}
        shadow-camera-far={40}
        shadow-camera-left={-10}
        shadow-camera-right={10}
        shadow-camera-top={10}
        shadow-camera-bottom={-10}
      />
      {/* Cyan key fill light */}
      <directionalLight position={[-10, 8, -6]} intensity={0.7} color="#38bdf8" />
      {/* Subtle purple rim backlight */}
      <directionalLight position={[4, 6, -10]} intensity={0.4} color="#a855f7" />

      {/* Orbit Controls bounded matching prototype */}
      <OrbitControls
        makeDefault
        minDistance={6}
        maxDistance={22}
        maxPolarAngle={Math.PI / 2.1}
        dampingFactor={0.05}
        enablePan={false}
      />

      {/* Dynamic Adaptive Diorama Pedestal Floor */}
      <Floor tenantsCount={tenants?.length ?? 0} booths={layout} />

      {/* Procedural Tenant Booths */}
      {layout.map((item) => (
        <Booth
          key={item.tenant.id}
          tenant={item.tenant}
          position={item.position}
          rotationY={item.rotationY}
          totalTenants={tenants?.length ?? 0}
          onSelect={onSelectTenant}
        />
      ))}

      {/* Low-Poly Visitor Agents */}
      <Avatars avatars={avatars} tenants={tenants} />
    </>
  );
}
