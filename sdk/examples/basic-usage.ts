/**
 * Basic usage examples for Cortex Memory SDK
 */

import { CortexClient } from "../src/index";

async function main() {
  // Initialize client
  const client = new CortexClient({
    baseUrl: "http://localhost:9123",
    apiKey: "your-api-key", // Optional
    appId: "myapp", // Optional: default for all requests
    externalUserId: "user123", // Optional: default for all requests
  });

  // Health check
  console.log("Checking health...");
  const health = await client.health();
  console.log("Health:", health);

  // Store a memory
  console.log("\nStoring memory...");
  const memory = await client.storeMemory({
    appId: "myapp",
    externalUserId: "user123",
    content: "Der Benutzer mag Kaffee mit Hafermilch",
    metadata: {
      source: "chat",
      timestamp: new Date().toISOString(),
    },
  });
  console.log("Memory stored:", memory);

  // Query memories
  console.log("\nQuerying memories...");
  const results = await client.queryMemory({
    appId: "myapp",
    externalUserId: "user123",
    query: "Was mag der Benutzer trinken?",
    limit: 5,
  });
  console.log("Query results:", results);
  results.forEach((result, index) => {
    console.log(
      `  ${index + 1}. ${result.content} (similarity: ${result.similarity.toFixed(2)})`
    );
  });

  // Create a bundle
  console.log("\nCreating bundle...");
  const bundle = await client.createBundle({
    appId: "myapp",
    externalUserId: "user123",
    name: "Coffee Preferences",
  });
  console.log("Bundle created:", bundle);

  // Store memory in bundle
  console.log("\nStoring memory in bundle...");
  const bundleMemory = await client.storeMemory({
    appId: "myapp",
    externalUserId: "user123",
    content: "Lieblingskaffee: Latte",
    bundleId: bundle.id,
  });
  console.log("Memory stored in bundle:", bundleMemory);

  // List bundles
  console.log("\nListing bundles...");
  const bundles = await client.listBundles("myapp", "user123");
  console.log("Bundles:", bundles);

  // Query memories in bundle
  console.log("\nQuerying memories in bundle...");
  const bundleResults = await client.queryMemory({
    appId: "myapp",
    externalUserId: "user123",
    query: "Kaffee",
    bundleId: bundle.id,
    limit: 10,
  });
  console.log("Bundle query results:", bundleResults);

  // Generate embeddings for existing memories
  console.log("\nGenerating embeddings...");
  const embeddingResult = await client.generateEmbeddings(10);
  console.log("Embedding generation:", embeddingResult);

  // Delete memory
  console.log("\nDeleting memory...");
  const deleteResult = await client.deleteMemory(
    memory.id,
    "myapp",
    "user123"
  );
  console.log("Memory deleted:", deleteResult);

  // Delete bundle
  console.log("\nDeleting bundle...");
  const deleteBundleResult = await client.deleteBundle(
    bundle.id,
    "myapp",
    "user123"
  );
  console.log("Bundle deleted:", deleteBundleResult);
}

// Run examples
main().catch((error) => {
  console.error("Error:", error);
  process.exit(1);
});
