# Cortex Memory SDK

TypeScript SDK for Cortex Memory API - Neutron-compatible memory management for OpenClaw agents.

## Installation

```bash
npm install @cortex/memory-sdk
# or
yarn add @cortex/memory-sdk
```

## Quick Start

```typescript
import { CortexClient } from "@cortex/memory-sdk";

const client = new CortexClient({
  baseUrl: "http://localhost:9123",
  apiKey: "your-api-key", // Optional
  appId: "myapp", // Optional: default for all requests
  externalUserId: "user123", // Optional: default for all requests
});

// Store a memory
const memory = await client.storeMemory({
  appId: "myapp",
  externalUserId: "user123",
  content: "Der Benutzer mag Kaffee",
  metadata: { source: "chat" },
});

// Query memories
const results = await client.queryMemory({
  appId: "myapp",
  externalUserId: "user123",
  query: "Was mag der Benutzer?",
  limit: 5,
});
```

## Features

- ✅ **Neutron-compatible API** - Same interface as Neutron Memory API
- ✅ **Dual parameter support** - Query parameters (Neutron-style) or Body parameters (Cortex-style)
- ✅ **TypeScript-first** - Full type safety
- ✅ **Bundle support** - Organize memories into logical groups
- ✅ **Semantic search** - Automatic embedding-based search
- ✅ **Embedding generation** - Batch generate embeddings for existing memories
- ✅ **Error handling** - Comprehensive error types with `CortexError`

## API Reference

### Client Configuration

```typescript
interface CortexClientConfig {
  baseUrl?: string; // Default: "http://localhost:9123"
  apiKey?: string; // Optional API key
  appId?: string; // Default appId for all requests
  externalUserId?: string; // Default externalUserId for all requests
}
```

### Methods

#### `storeMemory(request, options?)`

Store a memory (seed).

```typescript
await client.storeMemory({
  appId: "myapp",
  externalUserId: "user123",
  content: "Memory content",
  metadata: { key: "value" },
  bundleId: 1, // Optional
}, { useQueryParams: true }); // Optional: use Neutron-style query params
```

#### `queryMemory(request, options?)`

Query memories with semantic search.

```typescript
const results = await client.queryMemory({
  appId: "myapp",
  externalUserId: "user123",
  query: "search query",
  limit: 5,
  bundleId: 1, // Optional: filter by bundle
});
```

#### `deleteMemory(id, appId?, externalUserId?)`

Delete a memory.

```typescript
await client.deleteMemory(1, "myapp", "user123");
```

#### `createBundle(request, options?)`

Create a bundle for organizing memories.

```typescript
const bundle = await client.createBundle({
  appId: "myapp",
  externalUserId: "user123",
  name: "Bundle Name",
});
```

#### `listBundles(appId?, externalUserId?)`

List all bundles for a tenant.

```typescript
const bundles = await client.listBundles("myapp", "user123");
```

#### `getBundle(id, appId?, externalUserId?)`

Get a bundle by ID.

```typescript
const bundle = await client.getBundle(1, "myapp", "user123");
```

#### `deleteBundle(id, appId?, externalUserId?)`

Delete a bundle (memories remain, bundleId set to null).

```typescript
await client.deleteBundle(1, "myapp", "user123");
```

#### `generateEmbeddings(batchSize?)`

Generate embeddings for existing memories that don't have embeddings yet.

```typescript
const result = await client.generateEmbeddings(10); // Optional: batch size (default: 10, max: 100)
// Returns: { message: "Embeddings generation started" }
```

#### `health()`

Health check.

```typescript
const health = await client.health();
```

## Examples

See [examples/basic-usage.ts](examples/basic-usage.ts) for complete examples.

## Compatibility

- **Neutron Memory API**: Fully compatible API structure
- **Query Parameters**: Supports Neutron-style query parameters
- **Body Parameters**: Supports Cortex-style body parameters
- **Both formats**: Can use either format seamlessly

## License

MIT
