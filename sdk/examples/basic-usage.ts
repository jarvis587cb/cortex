/**
 * Basic usage examples for Cortex Memory SDK
 */

import { CortexClient } from "../src/client";

async function main() {
  // Initialize client
  const client = new CortexClient({
    baseUrl: "http://localhost:9123",
    apiKey: "your-api-key", // Optional
    appId: "myapp", // Optional: default for all requests
    externalUserId: "user123", // Optional: default for all requests
  });

  try {
    // Health check
    const health = await client.health();
    console.log("Health:", health);

    // Store a memory (using body parameters - Cortex style)
    const stored = await client.storeMemory({
      appId: "myapp",
      externalUserId: "user123",
      content: "Der Benutzer mag Kaffee mit Hafermilch",
      metadata: { source: "chat", type: "preference" },
    });
    console.log("Stored memory ID:", stored.id);

    // Store a memory (using query parameters - Neutron style)
    const stored2 = await client.storeMemory(
      {
        appId: "myapp",
        externalUserId: "user123",
        content: "Der Benutzer liest gerne Science-Fiction-Bücher",
        metadata: { source: "chat" },
      },
      { useQueryParams: true }
    );
    console.log("Stored memory ID:", stored2.id);

    // Create a bundle
    const bundle = await client.createBundle({
      appId: "myapp",
      externalUserId: "user123",
      name: "Coffee Preferences",
    });
    console.log("Created bundle ID:", bundle.id);

    // Store memory in bundle
    const storedInBundle = await client.storeMemory({
      appId: "myapp",
      externalUserId: "user123",
      content: "Lieblingskaffee: Latte mit Hafermilch",
      bundleId: bundle.id,
    });
    console.log("Stored in bundle:", storedInBundle.id);

    // Query memories
    const results = await client.queryMemory({
      appId: "myapp",
      externalUserId: "user123",
      query: "Was mag der Benutzer trinken?",
      limit: 5,
    });
    console.log("Query results:", results);

    // Query memories in bundle
    const bundleResults = await client.queryMemory({
      appId: "myapp",
      externalUserId: "user123",
      query: "Kaffee",
      bundleId: bundle.id,
      limit: 10,
    });
    console.log("Bundle query results:", bundleResults);

    // List bundles
    const bundles = await client.listBundles("myapp", "user123");
    console.log("Bundles:", bundles);

    // Delete a memory
    await client.deleteMemory(stored.id, "myapp", "user123");
    console.log("Memory deleted");

    // Delete a bundle
    await client.deleteBundle(bundle.id, "myapp", "user123");
    console.log("Bundle deleted");
  } catch (error) {
    console.error("Error:", error);
  }
}

// Run example if executed directly
if (require.main === module) {
  main().catch(console.error);
}
