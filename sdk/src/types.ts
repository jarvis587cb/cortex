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
