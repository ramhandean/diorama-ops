export interface Tenant {
  id: string;
  name: string;
  url: string;
  accent: string;
  logo_path?: string;
  use_favicon: boolean;
  position_x?: number;
  position_y?: number;
  position_z?: number;
  status: 'up' | 'degraded' | 'down';
  active: number;
}

export type AvatarState = 'walking' | 'idle' | 'leaving';

export interface AvatarItem {
  vid: string;
  tenantId: string;
  spawnTime: number;
  state: AvatarState;
  targetPos: [number, number, number];
  currentPos: [number, number, number];
}

export interface WSSnapshotTenant {
  id: string;
  name: string;
  active: number;
  status: string;
}

export interface WSEnterMessage {
  t: 'enter';
  tenant: string;
  vid: string;
}

export interface WSLeaveMessage {
  t: 'leave';
  tenant: string;
  vid: string;
}

export interface WSStatusMessage {
  t: 'status';
  tenant: string;
  state: 'up' | 'degraded' | 'down';
}

export interface WSSnapMessage {
  t: 'snap';
  tenants: WSSnapshotTenant[];
}

export type WSMessage = WSEnterMessage | WSLeaveMessage | WSStatusMessage | WSSnapMessage;
