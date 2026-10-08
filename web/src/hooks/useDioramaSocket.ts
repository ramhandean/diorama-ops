import { useEffect, useState, useRef, useCallback } from 'react';
import { Tenant, AvatarItem, WSMessage } from '../types';
import { computeBoothLayout } from '../scene/layout';

export function useDioramaSocket() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [avatars, setAvatars] = useState<AvatarItem[]>([]);
  const [connected, setConnected] = useState<boolean>(false);
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);

  const tenantsRef = useRef<Tenant[]>([]);
  tenantsRef.current = tenants;

  const avatarsRef = useRef<AvatarItem[]>([]);
  avatarsRef.current = avatars;

  // Initial fetch of tenants
  const fetchTenants = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/tenants');
      if (res.ok) {
        const data: Tenant[] = await res.json();
        setTenants(Array.isArray(data) ? data : []);
      }
    } catch {
      // Ignored, will retry
    }
  }, []);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  // Handle incoming websocket messages
  const handleMessage = useCallback((msg: WSMessage) => {
    switch (msg.t) {
      case 'snap': {
        const activeMap = new Map<string, { active: number; status: string }>();
        msg.tenants.forEach((st) => {
          activeMap.set(st.id, { active: st.active, status: st.status });
        });

        setTenants((prev) =>
          prev.map((t) => {
            const match = activeMap.get(t.id);
            if (match) {
              return {
                ...t,
                active: match.active,
                status: match.status as 'up' | 'degraded' | 'down',
              };
            }
            return t;
          })
        );
        break;
      }

      case 'enter': {
        // Increment tenant active counter
        setTenants((prev) =>
          prev.map((t) => (t.id === msg.tenant ? { ...t, active: t.active + 1 } : t))
        );

        // Find target booth position
        const layout = computeBoothLayout(tenantsRef.current);
        const booth = layout.find((b) => b.tenant.id === msg.tenant);
        const targetPos: [number, number, number] = booth
          ? [booth.position[0], 0.35, booth.position[2]]
          : [0, 0.35, 0];

        // Add slight random offset at booth entrance so avatars don't stack exactly on top
        const angle = Math.random() * Math.PI * 2;
        const dist = 0.5 + Math.random() * 0.7;
        const finalTarget: [number, number, number] = [
          targetPos[0] + Math.cos(angle) * dist,
          0.35,
          targetPos[2] + Math.sin(angle) * dist,
        ];

        // Cap at 60 active 3D avatars
        setAvatars((prev) => {
          if (prev.length >= 60) {
            return prev;
          }
          return [
            ...prev.filter((a) => a.vid !== msg.vid),
            {
              vid: msg.vid,
              tenantId: msg.tenant,
              spawnTime: performance.now(),
              state: 'walking',
              currentPos: [0, 0.35, 0],
              targetPos: finalTarget,
            },
          ];
        });
        break;
      }

      case 'leave': {
        // Decrement tenant active counter
        setTenants((prev) =>
          prev.map((t) =>
            t.id === msg.tenant ? { ...t, active: Math.max(0, t.active - 1) } : t
          )
        );

        // Mark avatar as leaving, then remove after animation
        setAvatars((prev) =>
          prev.map((a) => (a.vid === msg.vid ? { ...a, state: 'leaving' } : a))
        );

        setTimeout(() => {
          setAvatars((prev) => prev.filter((a) => a.vid !== msg.vid));
        }, 800);
        break;
      }

      case 'status': {
        setTenants((prev) =>
          prev.map((t) => (t.id === msg.tenant ? { ...t, status: msg.state } : t))
        );
        break;
      }
    }
  }, []);

  // WebSocket connection management with reconnect
  useEffect(() => {
    let ws: WebSocket | null = null;
    let sse: EventSource | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let isCancelled = false;

    function connect() {
      if (isCancelled) return;
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${proto}//${window.location.host}/ws?role=dashboard`;

      try {
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
          setConnected(true);
        };

        ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            handleMessage(data);
          } catch {
            // Non-JSON ignored
          }
        };

        ws.onclose = () => {
          setConnected(false);
          if (!isCancelled) {
            timer = setTimeout(connect, 3000);
          }
        };

        ws.onerror = () => {
          try {
            ws?.close();
          } catch {
            // Ignore
          }
        };
      } catch {
        // Fallback to Server-Sent Events (/events)
        try {
          sse = new EventSource('/events');
          sse.onopen = () => setConnected(true);
          sse.onmessage = (e) => {
            try {
              const data = JSON.parse(e.data);
              handleMessage(data);
            } catch {
              // Ignore
            }
          };
          sse.onerror = () => {
            setConnected(false);
            sse?.close();
            if (!isCancelled) timer = setTimeout(connect, 4000);
          };
        } catch {
          if (!isCancelled) timer = setTimeout(connect, 4000);
        }
      }
    }

    connect();

    return () => {
      isCancelled = true;
      if (timer) clearTimeout(timer);
      if (ws) ws.close();
      if (sse) sse.close();
    };
  }, [handleMessage]);

  return {
    tenants,
    avatars,
    connected,
    selectedTenant,
    setSelectedTenant,
    refetch: fetchTenants,
  };
}
