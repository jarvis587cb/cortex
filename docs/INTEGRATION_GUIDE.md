# Cross-Platform Integration Guide: Cortex

**Datum:** 2026-02-19  
**Ziel:** Integration von Cortex in verschiedene Plattformen (Discord, Slack, WhatsApp, Web)

## Executive Summary

Cortex ermöglicht **Cross-Platform Continuity** für OpenClaw Agents durch Multi-Tenant-Support und REST API. Dieser Guide zeigt, wie Cortex in verschiedene Plattformen integriert wird.

## Architektur-Übersicht

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Discord   │     │    Slack    │     │  WhatsApp   │
│    Bot      │     │     Bot     │     │     Bot     │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                  │                    │
       └──────────────────┼────────────────────┘
                          │
                          ▼
                  ┌──────────────┐
                  │   Cortex     │
                  │  REST API    │
                  │  (Port 9123) │
                  └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │   SQLite     │
                  │   Database   │
                  └──────────────┘
```

**Prinzip:**
- Jede Plattform nutzt dieselbe Cortex-Instanz
- Multi-Tenant-Isolation durch `appId` + `externalUserId`
- Gemeinsames Memory durch gleiche `externalUserId`

## Setup

### 1. Cortex Server starten

```bash
# Cortex Server starten
cd /path/to/cortex
go run cmd/server/main.go

# Server läuft auf http://localhost:9123
```

### 2. TypeScript SDK installieren

```bash
# Im Projekt-Verzeichnis
cd sdk
npm install
npm run build

# In anderen Projekten verwenden
npm install @openclaw/cortex-sdk
```

### 3. Environment Variables

```bash
# .env
CORTEX_API_URL=http://localhost:9123
```

## Discord Integration

### Setup

```bash
npm install discord.js @openclaw/cortex-sdk
```

### Code-Beispiel

```typescript
import { Client, GatewayIntentBits } from 'discord.js';
import { CortexClient } from '@openclaw/cortex-sdk';

const discordClient = new Client({
    intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMessages]
});

const cortex = new CortexClient({
    baseUrl: process.env.CORTEX_API_URL!,
    appId: 'discord-bot',
    externalUserId: 'discord-user-123' // Kann dynamisch sein
});

discordClient.on('messageCreate', async (message) => {
    if (message.author.bot) return;

    // Memory speichern
    await cortex.storeMemory({
        appId: 'discord-bot',
        externalUserId: message.author.id,
        content: `User ${message.author.username} sagte: ${message.content}`,
        metadata: {
            channel: message.channel.id,
            guild: message.guild?.id,
            timestamp: new Date().toISOString()
        }
    });

    // Kontext abfragen
    const context = await cortex.queryMemory({
        appId: 'discord-bot',
        externalUserId: message.author.id,
        query: message.content,
        limit: 5
    });

    // Antwort mit Kontext generieren
    if (context.length > 0) {
        const relevantMemory = context[0];
        await message.reply(`Ich erinnere mich: ${relevantMemory.content}`);
    }
});

discordClient.login(process.env.DISCORD_TOKEN);
```

### Multi-Channel Support

```typescript
// Verschiedene Channels = verschiedene Bundles
const channelBundle = await cortex.createBundle({
    appId: 'discord-bot',
    externalUserId: message.author.id,
    name: `channel-${message.channel.id}`
});

await cortex.storeMemory({
    appId: 'discord-bot',
    externalUserId: message.author.id,
    bundleId: channelBundle.id,
    content: message.content
});
```

## Slack Integration

### Setup

```bash
npm install @slack/bolt @openclaw/cortex-sdk
```

### Code-Beispiel

```typescript
import { App } from '@slack/bolt';
import { CortexClient } from '@openclaw/cortex-sdk';

const app = new App({
    token: process.env.SLACK_BOT_TOKEN,
    signingSecret: process.env.SLACK_SIGNING_SECRET
});

const cortex = new CortexClient({
    baseUrl: process.env.CORTEX_API_URL!,
    appId: 'slack-bot'
});

app.message(async ({ message, say, client }) => {
    const userId = message.user!;

    // Memory speichern
    await cortex.storeMemory({
        appId: 'slack-bot',
        externalUserId: userId,
        content: `User sagte: ${message.text}`,
        metadata: {
            channel: message.channel,
            thread: message.ts,
            timestamp: new Date().toISOString()
        }
    });

    // Kontext abfragen
    const context = await cortex.queryMemory({
        appId: 'slack-bot',
        externalUserId: userId,
        query: message.text!,
        limit: 5
    });

    // Antwort mit Kontext
    if (context.length > 0) {
        await say({
            text: `Ich erinnere mich: ${context[0].content}`,
            thread_ts: message.ts
        });
    }
});

(async () => {
    await app.start(process.env.PORT || 3000);
    console.log('Slack Bot läuft!');
})();
```

### Thread-Support

```typescript
// Threads = Bundles
const threadBundle = await cortex.createBundle({
    appId: 'slack-bot',
    externalUserId: userId,
    name: `thread-${message.thread_ts || message.ts}`
});

await cortex.storeMemory({
    appId: 'slack-bot',
    externalUserId: userId,
    bundleId: threadBundle.id,
    content: message.text!
});
```

## WhatsApp Integration

### Setup

```bash
npm install whatsapp-web.js @openclaw/cortex-sdk
```

### Code-Beispiel

```typescript
import { Client, LocalAuth } from 'whatsapp-web.js';
import { CortexClient } from '@openclaw/cortex-sdk';
import qrcode from 'qrcode-terminal';

const whatsappClient = new Client({
    authStrategy: new LocalAuth()
});

const cortex = new CortexClient({
    baseUrl: process.env.CORTEX_API_URL!,
    appId: 'whatsapp-bot'
});

whatsappClient.on('qr', (qr) => {
    qrcode.generate(qr, { small: true });
});

whatsappClient.on('ready', () => {
    console.log('WhatsApp Bot bereit!');
});

whatsappClient.on('message', async (message) => {
    const contact = await message.getContact();
    const userId = contact.id._serialized;

    // Memory speichern
    await cortex.storeMemory({
        appId: 'whatsapp-bot',
        externalUserId: userId,
        content: `User ${contact.pushname} sagte: ${message.body}`,
        metadata: {
            chat: message.from,
            timestamp: new Date().toISOString(),
            hasMedia: message.hasMedia
        }
    });

    // Kontext abfragen
    const context = await cortex.queryMemory({
        appId: 'whatsapp-bot',
        externalUserId: userId,
        query: message.body,
        limit: 5
    });

    // Antwort mit Kontext
    if (context.length > 0) {
        await message.reply(`Ich erinnere mich: ${context[0].content}`);
    }
});

whatsappClient.initialize();
```

### Chat-Gruppen Support

```typescript
// Gruppen = Bundles
const groupBundle = await cortex.createBundle({
    appId: 'whatsapp-bot',
    externalUserId: userId,
    name: `group-${message.from}`
});

await cortex.storeMemory({
    appId: 'whatsapp-bot',
    externalUserId: userId,
    bundleId: groupBundle.id,
    content: message.body
});
```

## Web Interface Integration

### Setup

```bash
npm install express @openclaw/cortex-sdk
```

### Code-Beispiel

```typescript
import express from 'express';
import { CortexClient } from '@openclaw/cortex-sdk';

const app = express();
app.use(express.json());

const cortex = new CortexClient({
    baseUrl: process.env.CORTEX_API_URL!,
    appId: 'web-interface'
});

// Memory speichern
app.post('/api/memory', async (req, res) => {
    const { userId, content, metadata } = req.body;

    const memory = await cortex.storeMemory({
        appId: 'web-interface',
        externalUserId: userId,
        content,
        metadata
    });

    res.json(memory);
});

// Memory abfragen
app.post('/api/memory/query', async (req, res) => {
    const { userId, query, limit } = req.body;

    const results = await cortex.queryMemory({
        appId: 'web-interface',
        externalUserId: userId,
        query,
        limit: limit || 10
    });

    res.json(results);
});

// Memory-Liste
app.get('/api/memory/:userId', async (req, res) => {
    const { userId } = req.params;

    // Nutze Query mit leerem String für alle Memories
    const results = await cortex.queryMemory({
        appId: 'web-interface',
        externalUserId: userId,
        query: '',
        limit: 100
    });

    res.json(results);
});

app.listen(3000, () => {
    console.log('Web Interface läuft auf Port 3000');
});
```

### React Frontend Beispiel

```typescript
// hooks/useCortex.ts
import { useState } from 'react';
import { CortexClient } from '@openclaw/cortex-sdk';

const cortex = new CortexClient({
    baseUrl: 'http://localhost:9123',
    appId: 'web-interface'
});

export function useCortex(userId: string) {
    const [memories, setMemories] = useState([]);

    const storeMemory = async (content: string) => {
        await cortex.storeMemory({
            appId: 'web-interface',
            externalUserId: userId,
            content
        });
    };

    const queryMemory = async (query: string) => {
        const results = await cortex.queryMemory({
            appId: 'web-interface',
            externalUserId: userId,
            query,
            limit: 10
        });
        setMemories(results);
        return results;
    };

    return { memories, storeMemory, queryMemory };
}
```

## Cross-Platform Continuity

### Gemeinsames Memory über Plattformen

**Prinzip:** Gleiche `externalUserId` = gemeinsames Memory

```typescript
// Discord Bot
const discordCortex = new CortexClient({
    baseUrl: 'http://localhost:9123',
    appId: 'discord-bot',
    externalUserId: 'user-123' // Gleiche ID!
});

// Slack Bot
const slackCortex = new CortexClient({
    baseUrl: 'http://localhost:9123',
    appId: 'slack-bot',
    externalUserId: 'user-123' // Gleiche ID!
});

// Beide können auf gemeinsames Memory zugreifen
const discordMemory = await discordCortex.storeMemory({
    appId: 'discord-bot',
    externalUserId: 'user-123',
    content: "User mag Kaffee"
});

// Slack kann dasselbe Memory abfragen
const slackContext = await slackCortex.queryMemory({
    appId: 'slack-bot',
    externalUserId: 'user-123',
    query: "Kaffee",
    limit: 5
});
// Findet das Memory von Discord!
```

### User-ID Mapping

**Problem:** Verschiedene Plattformen haben verschiedene User-IDs

**Lösung:** Mapping-Tabelle oder einheitliche externe ID

```typescript
// Mapping-Tabelle
const userMapping = {
    'discord:123456789': 'user-123',
    'slack:U123456': 'user-123',
    'whatsapp:+49123456789': 'user-123'
};

function getExternalUserId(platform: string, platformUserId: string): string {
    const key = `${platform}:${platformUserId}`;
    return userMapping[key] || `unknown-${platform}-${platformUserId}`;
}

// Verwendung
const externalUserId = getExternalUserId('discord', message.author.id);
await cortex.storeMemory({
    appId: 'discord-bot',
    externalUserId,
    content: message.content
});
```

## Webhooks für Cross-Platform Sync

### Webhook Setup

```typescript
// Webhook-Endpoint für Cross-Platform-Sync
app.post('/webhook/cortex', async (req, res) => {
    const signature = req.headers['x-cortex-signature'];
    const payload = JSON.stringify(req.body);

    // Signatur verifizieren
    const isValid = verifyWebhookSignature(payload, signature);
    if (!isValid) {
        return res.status(401).json({ error: 'Invalid signature' });
    }

    const { event, data } = req.body;

    if (event === 'memory.created') {
        // Memory wurde erstellt - synchronisiere mit anderen Plattformen
        await syncToOtherPlatforms(data);
    }

    res.json({ success: true });
});

async function syncToOtherPlatforms(memory: any) {
    // Beispiel: Sende Notification an Slack
    await slackClient.chat.postMessage({
        channel: '#cortex-updates',
        text: `Neues Memory: ${memory.content}`
    });
}
```

## Best Practices

### 1. Tenant-Isolation

```typescript
// Immer appId + externalUserId verwenden
await cortex.storeMemory({
    appId: 'discord-bot',        // Plattform-ID
    externalUserId: userId,      // User-ID
    content: message.content
});
```

### 2. Error Handling

```typescript
try {
    await cortex.storeMemory({...});
} catch (error) {
    console.error('Cortex-Fehler:', error);
    // Fallback-Verhalten
}
```

### 3. Rate Limiting

```typescript
// Cortex hat eingebautes Rate Limiting
// Bei zu vielen Requests: 429 Too Many Requests
// Implementiere Retry-Logik
async function storeWithRetry(data: any, retries = 3) {
    for (let i = 0; i < retries; i++) {
        try {
            return await cortex.storeMemory(data);
        } catch (error) {
            if (error.status === 429 && i < retries - 1) {
                await sleep(1000 * (i + 1)); // Exponential backoff
                continue;
            }
            throw error;
        }
    }
}
```

### 4. Bundles für Organisation

```typescript
// Nutze Bundles für logische Gruppierung
const conversationBundle = await cortex.createBundle({
    appId: 'discord-bot',
    externalUserId: userId,
    name: `conversation-${channelId}`
});

await cortex.storeMemory({
    appId: 'discord-bot',
    externalUserId: userId,
    bundleId: conversationBundle.id,
    content: message.content
});
```

## Troubleshooting

### Problem: Memory wird nicht gefunden

**Lösung:** Prüfe `appId` und `externalUserId`

```typescript
// Falsch: Verschiedene IDs
await cortex.storeMemory({
    appId: 'discord-bot',
    externalUserId: 'user-123',
    content: "Test"
});

await cortex.queryMemory({
    appId: 'discord-bot',
    externalUserId: 'user-456', // Falsch!
    query: "Test"
});

// Richtig: Gleiche IDs
await cortex.queryMemory({
    appId: 'discord-bot',
    externalUserId: 'user-123', // Richtig!
    query: "Test"
});
```

### Problem: Performance bei vielen Memories

**Lösung:** Nutze Bundles und Limits

```typescript
// Begrenze Query-Ergebnisse
const results = await cortex.queryMemory({
    appId: 'discord-bot',
    externalUserId: userId,
    query: message.content,
    limit: 5 // Nur Top-5 Ergebnisse
});

// Nutze Bundles für Filterung
const bundleResults = await cortex.queryMemory({
    appId: 'discord-bot',
    externalUserId: userId,
    bundleId: conversationBundle.id,
    query: message.content,
    limit: 10
});
```

## Fazit

Cortex ermöglicht **nahtlose Cross-Platform Continuity** durch:

- ✅ **Multi-Tenant-Support:** Isolierte Memory pro Plattform/User
- ✅ **REST API:** Plattform-unabhängiger Zugriff
- ✅ **TypeScript SDK:** Einfache Integration
- ✅ **Bundles:** Logische Organisation von Memories
- ✅ **Webhooks:** Event-basierte Synchronisation

**Mit Cortex können OpenClaw Agents über alle Plattformen hinweg konsistentes Memory behalten.**

---

**Siehe auch:**
- [CORTEX_NEUTRON_ALTERNATIVE.md](CORTEX_NEUTRON_ALTERNATIVE.md) - Feature-Vergleich
- [API.md](API.md) - Vollständige API-Dokumentation
- [README.md](README.md) - Projekt-Übersicht
