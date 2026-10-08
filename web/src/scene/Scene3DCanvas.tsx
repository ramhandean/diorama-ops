import { Canvas } from '@react-three/fiber';
import { DioramaScene } from './DioramaScene';
import { Tenant, AvatarItem } from '../types';

interface Scene3DCanvasProps {
  tenants: Tenant[];
  avatars: AvatarItem[];
  onSelectTenant: (tenant: Tenant) => void;
}

export default function Scene3DCanvas({
  tenants,
  avatars,
  onSelectTenant,
}: Scene3DCanvasProps) {
  return (
    <Canvas
      shadows
      camera={{ position: [11, 10, 11], fov: 40 }}
      gl={{
        antialias: true,
        alpha: false,
        powerPreference: 'high-performance',
      }}
    >
      <color attach="background" args={['#0c0d14']} />
      <DioramaScene
        tenants={tenants}
        avatars={avatars}
        onSelectTenant={onSelectTenant}
      />
    </Canvas>
  );
}
