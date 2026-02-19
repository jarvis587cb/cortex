/**
 * TypeScript Types for Cortex Memory API
 * Neutron-compatible types
 */

export interface StoreMemoryRequest {
  appId: string;
  externalUserId: string;
  content: string;
  metadata?: Record<string, any>;
  bundleId?: number;
}

export interface StoreMemoryResponse {
  id: number;
  message: string;
}

export interface QueryMemoryRequest {
  appId: string;
  externalUserId: string;
  query: string;
  limit?: number;
  bundleId?: number;
  /** 0-1: only return results with similarity >= threshold (default 0 = no filter) */
  threshold?: number;
  /** optional: limit search to these memory IDs */
  seedIds?: number[];
}

export interface QueryMemoryResult {
  id: number;
  content: string;
  metadata: Record<string, any>;
  created_at: string;
  similarity: number;
}

export interface DeleteMemoryResponse {
  message: string;
  id: number;
}

export interface CreateBundleRequest {
  appId: string;
  externalUserId: string;
  name: string;
}

export interface BundleResponse {
  id: number;
  name: string;
  app_id: string;
  external_user_id: string;
  created_at: string;
}

export interface CortexClientConfig {
  baseUrl?: string;
  /** Optional; when CORTEX_API_KEY is set on the server, send via X-API-Key header */
  apiKey?: string;
  appId?: string;
  externalUserId?: string;
}

export interface GenerateEmbeddingsResponse {
  message: string;
}

export class CortexError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public response?: any
  ) {
    super(message);
    this.name = "CortexError";
  }
}
