/**
 * Tests for CortexClient
 * 
 * Note: These are example tests. For full test coverage,
 * install jest or vitest and run: npm test
 */

import { CortexClient, CortexError } from "./index";

describe("CortexClient", () => {
  const client = new CortexClient({
    baseUrl: "http://localhost:9123",
    apiKey: "test-key",
    appId: "test-app",
    externalUserId: "test-user",
  });

  describe("health", () => {
    it("should return health status", async () => {
      const health = await client.health();
      expect(health).toHaveProperty("status");
      expect(health).toHaveProperty("timestamp");
      expect(health.status).toBe("ok");
    });
  });

  describe("storeMemory", () => {
    it("should store a memory", async () => {
      const result = await client.storeMemory({
        appId: "test-app",
        externalUserId: "test-user",
        content: "Test memory",
        metadata: { source: "test" },
      });

      expect(result).toHaveProperty("id");
      expect(result).toHaveProperty("message");
      expect(result.message).toBe("Memory stored successfully");
    });

    it("should support query parameters", async () => {
      const result = await client.storeMemory(
        {
          appId: "test-app",
          externalUserId: "test-user",
          content: "Test memory",
        },
        { useQueryParams: true }
      );

      expect(result).toHaveProperty("id");
    });
  });

  describe("queryMemory", () => {
    it("should query memories", async () => {
      const results = await client.queryMemory({
        appId: "test-app",
        externalUserId: "test-user",
        query: "test",
        limit: 5,
      });

      expect(Array.isArray(results)).toBe(true);
      if (results.length > 0) {
        expect(results[0]).toHaveProperty("id");
        expect(results[0]).toHaveProperty("content");
        expect(results[0]).toHaveProperty("similarity");
      }
    });
  });

  describe("deleteMemory", () => {
    it("should delete a memory", async () => {
      // First create a memory
      const created = await client.storeMemory({
        appId: "test-app",
        externalUserId: "test-user",
        content: "Memory to delete",
      });

      // Then delete it
      const result = await client.deleteMemory(
        created.id,
        "test-app",
        "test-user"
      );

      expect(result).toHaveProperty("message");
      expect(result).toHaveProperty("id");
      expect(result.id).toBe(created.id);
    });
  });

  describe("bundles", () => {
    it("should create a bundle", async () => {
      const bundle = await client.createBundle({
        appId: "test-app",
        externalUserId: "test-user",
        name: "Test Bundle",
      });

      expect(bundle).toHaveProperty("id");
      expect(bundle).toHaveProperty("name");
      expect(bundle.name).toBe("Test Bundle");
    });

    it("should list bundles", async () => {
      const bundles = await client.listBundles("test-app", "test-user");
      expect(Array.isArray(bundles)).toBe(true);
    });

    it("should get a bundle", async () => {
      const created = await client.createBundle({
        appId: "test-app",
        externalUserId: "test-user",
        name: "Test Bundle 2",
      });

      const bundle = await client.getBundle(
        created.id,
        "test-app",
        "test-user"
      );

      expect(bundle.id).toBe(created.id);
      expect(bundle.name).toBe("Test Bundle 2");
    });

    it("should delete a bundle", async () => {
      const created = await client.createBundle({
        appId: "test-app",
        externalUserId: "test-user",
        name: "Bundle to delete",
      });

      const result = await client.deleteBundle(
        created.id,
        "test-app",
        "test-user"
      );

      expect(result).toHaveProperty("message");
      expect(result).toHaveProperty("id");
      expect(result.id).toBe(created.id);
    });
  });

  describe("generateEmbeddings", () => {
    it("should generate embeddings", async () => {("should generate embeddings", async () => {
      const result = await client.generateEmbeddings(10);
      expect(result).toHaveProperty("message");
    });

    it("should support custom batch size", async () => {
      const result = await client.generateEmbeddings(20);
      expect(result).toHaveProperty("message");
    });
  });

  describe("error handling", () => {
    it("should throw CortexError on API errors", async () => {
      const invalidClient = new CortexClient({
        baseUrl: "http://localhost:9123",
        apiKey: "invalid-key",
      });

      await expect(
        invalidClient.storeMemory({
          appId: "test",
          externalUserId: "test",
          content: "test",
        })
      ).rejects.toThrow(CortexError);
    });
  });
});
