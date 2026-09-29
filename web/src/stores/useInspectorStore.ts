import { create } from 'zustand';
import { HTTPTransaction, RequestSummary, TunnelInfo } from '../types';

interface InspectorState {
  requests: RequestSummary[];
  total: number;
  selectedId: string | null;
  selectedTransaction: HTTPTransaction | null;
  isLoadingDetail: boolean;
  isPaused: boolean;
  searchQuery: string;
  methodFilter: string;
  statusFilter: string;
  tunnel: TunnelInfo;
  wsConnected: boolean;

  // Actions
  setWsConnected: (connected: boolean) => void;
  setTunnel: (tunnel: Partial<TunnelInfo>) => void;
  setPaused: (paused: boolean) => void;
  togglePaused: () => void;
  setSearchQuery: (query: string) => void;
  setMethodFilter: (method: string) => void;
  setStatusFilter: (status: string) => void;
  addRequest: (req: RequestSummary) => void;
  setRequests: (requests: RequestSummary[], total: number) => void;
  selectRequest: (id: string | null) => Promise<void>;
  fetchRequests: () => Promise<void>;
  fetchStatus: () => Promise<void>;
  clearRequests: () => Promise<void>;
  replayRequest: (id: string, overrideBody?: string) => Promise<HTTPTransaction | null>;
}

export const useInspectorStore = create<InspectorState>((set, get) => ({
  requests: [],
  total: 0,
  selectedId: null,
  selectedTransaction: null,
  isLoadingDetail: false,
  isPaused: false,
  searchQuery: '',
  methodFilter: 'ALL',
  statusFilter: 'ALL',
  tunnel: {
    url: '',
    status: 'starting',
    target: 'http://localhost:8000',
  },
  wsConnected: false,

  setWsConnected: (connected) => set({ wsConnected: connected }),
  
  setTunnel: (partial) => set((state) => ({ tunnel: { ...state.tunnel, ...partial } })),

  setPaused: (paused) => set({ isPaused: paused }),
  
  togglePaused: () => set((state) => ({ isPaused: !state.isPaused })),

  setSearchQuery: (searchQuery) => set({ searchQuery }),

  setMethodFilter: (methodFilter) => set({ methodFilter }),

  setStatusFilter: (statusFilter) => set({ statusFilter }),

  addRequest: (req) => {
    const { isPaused, requests, total } = get();
    if (isPaused) return;

    // Check if duplicate
    const exists = requests.some((r) => r.id === req.id);
    if (exists) {
      set({
        requests: requests.map((r) => (r.id === req.id ? req : r)),
      });
      return;
    }

    set({
      requests: [req, ...requests],
      total: total + 1,
    });
  },

  setRequests: (requests, total) => set({ requests, total }),

  selectRequest: async (id) => {
    if (!id) {
      set({ selectedId: null, selectedTransaction: null });
      return;
    }

    set({ selectedId: id, isLoadingDetail: true });
    try {
      const res = await fetch(`/api/requests/${encodeURIComponent(id)}`);
      if (!res.ok) throw new Error('Failed to load transaction');
      const data: HTTPTransaction = await res.json();
      set({ selectedTransaction: data, isLoadingDetail: false });
    } catch (err) {
      console.error(err);
      set({ isLoadingDetail: false });
    }
  },

  fetchRequests: async () => {
    try {
      const res = await fetch('/api/requests?limit=100');
      if (!res.ok) throw new Error('Failed to fetch requests');
      const data = await res.json();
      set({
        requests: data.requests || [],
        total: data.total || 0,
      });

      // Auto-select first request if none selected
      const currentSelected = get().selectedId;
      if (!currentSelected && data.requests && data.requests.length > 0) {
        get().selectRequest(data.requests[0].id);
      }
    } catch (err) {
      console.error('Failed to load requests:', err);
    }
  },

  fetchStatus: async () => {
    try {
      const res = await fetch('/api/status');
      if (!res.ok) return;
      const data = await res.json();
      set((state) => ({
        tunnel: {
          url: data.tunnel_url || state.tunnel.url,
          status: data.tunnel_status || state.tunnel.status,
          target: data.app_target || state.tunnel.target,
        },
      }));
    } catch (err) {
      console.error('Failed to load status:', err);
    }
  },

  clearRequests: async () => {
    try {
      await fetch('/api/requests', { method: 'DELETE' });
      set({
        requests: [],
        total: 0,
        selectedId: null,
        selectedTransaction: null,
      });
    } catch (err) {
      console.error('Failed to clear requests:', err);
    }
  },

  replayRequest: async (id: string, overrideBody?: string) => {
    try {
      const res = await fetch(`/api/requests/${encodeURIComponent(id)}/replay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: overrideBody !== undefined ? JSON.stringify({ body: overrideBody }) : undefined,
      });
      if (!res.ok) throw new Error('Replay failed');
      const newTx: HTTPTransaction = await res.json();
      
      // Select the newly replayed transaction
      get().selectRequest(newTx.id);
      return newTx;
    } catch (err) {
      console.error('Replay failed:', err);
      return null;
    }
  },
}));
