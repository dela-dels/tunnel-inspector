export interface RequestSummary {
  id: string;
  timestamp: string;
  method: string;
  url: string;
  path: string;
  status: number;
  duration: number; // in ms
  request_size: number;
  response_size: number;
}

export interface RequestData {
  method: string;
  url: string;
  path: string;
  query: Record<string, string[]>;
  headers: Record<string, string[]>;
  body: string;
  is_binary?: boolean;
  size: number;
}

export interface ResponseData {
  status_code: number;
  headers: Record<string, string[]>;
  body: string;
  is_binary?: boolean;
  size: number;
}

export interface HTTPTransaction {
  id: string;
  timestamp: string;
  duration: number;
  request: RequestData;
  response: ResponseData;
  created_at: string;
}

export interface TunnelInfo {
  url: string;
  status: 'starting' | 'connected' | 'stopped' | 'error';
  target: string;
}

export interface StatusInfo {
  status: string;
  app_target: string;
  tunnel_url: string;
  tunnel_status: string;
  dashboard_url: string;
  proxy_url: string;
  request_count: number;
}

export interface WebSocketEvent {
  type: string;
  request?: RequestSummary;
  tunnel?: {
    status: string;
    url: string;
  };
  id?: string;
  status?: string;
  url?: string;
}
