/**
 * HTTP Client for Cortex Memory API
 */

import {
  StoreMemoryRequest,
  StoreMemoryResponse,
  QueryMemoryRequest,
  QueryMemoryResult,
  DeleteMemoryResponse,
  CreateBundleRequest,
  BundleResponse,
  CortexClientConfig,
  CortexError,
  GenerateEmbeddingsResponse,
} from "./types";

export class CortexClient {
  private baseUrl: string;
  private defaultAppId?: string;
  private defaultExternalUserId?: string;

  constructor(config: CortexClientConfig = {}) {
    this.baseUrl = config.baseUrl || "http://localhost:9123";
    this.defaultAppId = config.appId;
    this.defaultExternalUserId = config.externalUserId;
  }

  private async request<T>(
    method: string,
    path: string,
    options: {
      body?: any;
      queryParams?: Record<string, string | number | undefined>;
    } = {}
  ): Promise<T> {
    const url = new URL(path, this.baseUrl);
    
    if (options.queryParams) {
      for (const [key, value] of Object.entries(options.queryParams)) {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value));
        }
      }
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    const fetchOptions: RequestInit = {
      method,
      headers,
    };

    if (options.body && (method === "POST" || method === "PUT")) {
      fetchOptions.body = JSON.stringify(options.body);
    }

    try {
      const response = await fetch(url.toString(), fetchOptions);
      const text = await response.text();
      
      if (!response.ok) {
        let errorBody: any;
        try {
          errorBody = JSON.parse(text);
        } catch {
          errorBody = { message: text };
        }
        throw new CortexError(
          errorBody.message || `HTTP ${response.status}`,
          response.status,
          errorBody
        );
      }

      if (text === "") {
        return {} as T;
      }

      return JSON.parse(text) as T;
    } catch (error) {
      if (error instanceof CortexError) {
        throw error;
      }
      throw new CortexError(
        `Network error: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  async storeMemory(
    request: StoreMemoryRequest,
    options?: { useQueryParams?: boolean }
  ): Promise<StoreMemoryResponse> {
    const useQueryParams = options?.useQueryParams ?? false;
    
    if (useQueryParams) {
      return this.request<StoreMemoryResponse>("POST", "/seeds", {
        queryParams: {
          appId: request.appId || this.defaultAppId,
          externalUserId: request.externalUserId || this.defaultExternalUserId,
        },
        body: {
          content: request.content,
          metadata: request.metadata,
          bundleId: request.bundleId,
        },
      });
    } else {
      return this.request<StoreMemoryResponse>("POST", "/seeds", {
        body: {
          appId: request.appId || this.defaultAppId,
          externalUserId: request.externalUserId || this.defaultExternalUserId,
          content: request.content,
          metadata: request.metadata,
          bundleId: request.bundleId,
        },
      });
    }
  }

  async queryMemory(
    request: QueryMemoryRequest,
    options?: { useQueryParams?: boolean }
  ): Promise<QueryMemoryResult[]> {
    const useQueryParams = options?.useQueryParams ?? false;
    
    if (useQueryParams) {
      return this.request<QueryMemoryResult[]>("POST", "/seeds/query", {
        queryParams: {
          appId: request.appId || this.defaultAppId,
          externalUserId: request.externalUserId || this.defaultExternalUserId,
        },
        body: {
          query: request.query,
          limit: request.limit,
          bundleId: request.bundleId,
        },
      });
    } else {
      return this.request<QueryMemoryResult[]>("POST", "/seeds/query", {
        body: {
          appId: request.appId || this.defaultAppId,
          externalUserId: request.externalUserId || this.defaultExternalUserId,
          query: request.query,
          limit: request.limit,
          bundleId: request.bundleId,
        },
      });
    }
  }

  async deleteMemory(
    id: number,
    appId?: string,
    externalUserId?: string
  ): Promise<DeleteMemoryResponse> {
    return this.request<DeleteMemoryResponse>("DELETE", `/seeds/${id}`, {
      queryParams: {
        appId: appId || this.defaultAppId,
        externalUserId: externalUserId || this.defaultExternalUserId,
      },
    });
  }

  async createBundle(
    request: CreateBundleRequest,
    options?: { useQueryParams?: boolean }
  ): Promise<BundleResponse> {
    const useQueryParams = options?.useQueryParams ?? false;
    
    if (useQueryParams) {
      return this.request<BundleResponse>("POST", "/bundles", {
        queryParams: {
          appId: request.appId || this.defaultAppId,
          externalUserId: request.externalUserId || this.defaultExternalUserId,
        },
        body: {
          name: request.name,
        },
      });
    } else {
      return this.request<BundleResponse>("POST", "/bundles", {
        body: request,
      });
    }
  }

  async listBundles(
    appId?: string,
    externalUserId?: string
  ): Promise<BundleResponse[]> {
    return this.request<BundleResponse[]>("GET", "/bundles", {
      queryParams: {
        appId: appId || this.defaultAppId,
        externalUserId: externalUserId || this.defaultExternalUserId,
      },
    });
  }

  async getBundle(
    id: number,
    appId?: string,
    externalUserId?: string
  ): Promise<BundleResponse> {
    return this.request<BundleResponse>("GET", `/bundles/${id}`, {
      queryParams: {
        appId: appId || this.defaultAppId,
        externalUserId: externalUserId || this.defaultExternalUserId,
      },
    });
  }

  async deleteBundle(
    id: number,
    appId?: string,
    externalUserId?: string
  ): Promise<{ message: string; id: number }> {
    return this.request<{ message: string; id: number }>(
      "DELETE",
      `/bundles/${id}`,
      {
        queryParams: {
          appId: appId || this.defaultAppId,
          externalUserId: externalUserId || this.defaultExternalUserId,
        },
      }
    );
  }

  async generateEmbeddings(
    batchSize?: number
  ): Promise<GenerateEmbeddingsResponse> {
    return this.request<GenerateEmbeddingsResponse>(
      "POST",
      "/seeds/generate-embeddings",
      {
        queryParams:
          batchSize !== undefined ? { batchSize } : undefined,
      }
    );
  }

  async health(): Promise<{ status: string; timestamp: string }> {
    return this.request<{ status: string; timestamp: string }>(
      "GET",
      "/health"
    );
  }
}
