const API_BASE = (import.meta.env.VITE_API_URL as string) || '';

function getApiKey(): string | null {
  return localStorage.getItem('cortex_api_key');
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };
  const key = getApiKey();
  if (key) headers['X-API-Key'] = key;
  const url = path.startsWith('http') ? path : `${API_BASE}${path}`;
  const res = await fetch(url, { ...options, headers });
  if (!res.ok) throw new Error(await res.text() || res.statusText);
  if (res.status === 204) return undefined as T;
  return res.json();
}

export interface Stats {
  memories: number;
  entities: number;
  relations: number;
}

export interface AnalyticsData {
  tenant_id: string;
  app_id: string;
  external_user_id: string;
  total_memories: number;
  total_bundles: number;
  memories_with_embeddings: number;
  memories_by_type: Record<string, number>;
  recent_activity: { type: string; id: number; timestamp: string }[];
  storage_stats: { memories_count: number; bundles_count: number };
  time_range: { start: string; end: string };
}

export interface Memory {
  id: number;
  content: string;
  type: string;
  created_at: string;
  metadata?: Record<string, unknown>;
}

export const api = {
  getStats: () => request<Stats>('/stats'),
  getAnalytics: (appId: string, externalUserId: string, days = 30) =>
    request<AnalyticsData>(`/analytics?appId=${encodeURIComponent(appId)}&externalUserId=${encodeURIComponent(externalUserId)}&days=${days}`),
  listSeeds: (appId: string, externalUserId: string, limit = 50, offset = 0) =>
    request<Memory[]>(`/seeds?appId=${encodeURIComponent(appId)}&externalUserId=${encodeURIComponent(externalUserId)}&limit=${limit}&offset=${offset}`),
  deleteSeed: (id: number, appId: string, externalUserId: string) =>
    request<{ message: string }>(`/seeds/${id}?appId=${encodeURIComponent(appId)}&externalUserId=${encodeURIComponent(externalUserId)}`, { method: 'DELETE' }),
};

export function getTenant(): { appId: string; externalUserId: string } {
  const appId = localStorage.getItem('cortex_app_id') || 'openclaw';
  const externalUserId = localStorage.getItem('cortex_external_user_id') || 'default';
  return { appId, externalUserId };
}

export function setTenant(appId: string, externalUserId: string): void {
  localStorage.setItem('cortex_app_id', appId);
  localStorage.setItem('cortex_external_user_id', externalUserId);
}
