import { Tenant } from '../types';

export interface LayoutBooth {
  tenant: Tenant;
  position: [number, number, number];
  rotationY: number;
}

export interface PlatformDimensions {
  boothRadius: number;
  floorRadius: number;
  segments: number;
}

export function computePlatformDimensions(count: number): PlatformDimensions {
  if (count <= 0) return { boothRadius: 0, floorRadius: 4.8, segments: 6 };
  if (count === 1) return { boothRadius: 2.2, floorRadius: 5.2, segments: 6 };
  if (count === 2) return { boothRadius: 2.8, floorRadius: 5.8, segments: 6 };
  if (count === 3) return { boothRadius: 3.2, floorRadius: 6.2, segments: 3 };
  if (count === 4) return { boothRadius: 3.6, floorRadius: 6.8, segments: 8 };
  if (count === 5) return { boothRadius: 4.0, floorRadius: 7.2, segments: 5 };
  if (count === 6) return { boothRadius: 4.4, floorRadius: 7.8, segments: 6 };

  const rBooth = Math.max(4.5, 2.0 + count * 0.52);
  const segments = count <= 12 ? count : 32;
  return { boothRadius: rBooth, floorRadius: rBooth + 2.5, segments };
}

export function computeBoothLayout(tenants: Tenant[] = []): LayoutBooth[] {
  const safeTenants = tenants || [];
  const count = safeTenants.length;
  if (count === 0) return [];

  const { boothRadius } = computePlatformDimensions(count);

  return safeTenants.map((tenant, index) => {
    // If manual coordinate is defined, use it
    if (
      tenant.position_x !== undefined &&
      tenant.position_x !== null &&
      tenant.position_z !== undefined &&
      tenant.position_z !== null
    ) {
      const posX = tenant.position_x;
      const posY = tenant.position_y ?? 0;
      const posZ = tenant.position_z;
      const rotY = Math.atan2(-posX, -posZ);
      return {
        tenant,
        position: [posX, posY, posZ],
        rotationY: rotY,
      };
    }

    // Dynamic radial polygon for N tenants
    const angle = (index / count) * Math.PI * 2 - Math.PI / 2;
    const posX = Math.cos(angle) * boothRadius;
    const posZ = Math.sin(angle) * boothRadius;
    const rotY = Math.atan2(-posX, -posZ);

    return {
      tenant,
      position: [posX, 0, posZ],
      rotationY: rotY,
    };
  });
}
