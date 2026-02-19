# Embeddings & Multimodal Implementation Plan

## Ziel

Implementierung von semantischer Suche mit Embeddings und Multimodal-Support für Cortex, kompatibel mit Neutron Memory API.

## Architektur-Entscheidung

### Option 1: HTTP-API zu Embedding-Service (Empfohlen)
**Vorteile:**
- ✅ Keine großen Dependencies
- ✅ Unterstützt Jina v4 (wie Neutron)
- ✅ Multimodal-Support out-of-the-box
- ✅ Leichtgewichtig für Cortex

**Nachteile:**
- ⚠️ Externe API-Abhängigkeit
- ⚠️ Netzwerk-Latenz

### Option 2: Go-native Embedding-Library
**Vorteile:**
- ✅ Keine externe Abhängigkeit
- ✅ Offline-Funktionalität

**Nachteile:**
- ⚠️ Größere Binary-Größe
- ⚠️ Begrenzte Multimodal-Unterstützung

### Option 3: SQLite-VSS Extension
**Vorteile:**
- ✅ Vektor-Suche in SQLite
- ✅ Keine externe Datenbank nötig

**Nachteile:**
- ⚠️ CGO erforderlich (nicht pure-Go)
- ⚠️ Komplexere Installation

## Gewählte Lösung: Hybrid-Ansatz

**Phase 1:** HTTP-API zu Embedding-Service (Jina API oder lokaler Service)
**Phase 2:** Optional: Lokale Embedding-Library als Fallback

## Implementierungs-Schritte

### 1. Embedding-Service Integration
- HTTP-Client für Embedding-Generierung
- Unterstützung für Text, Bilder, Dokumente
- Caching von Embeddings

### 2. Datenbank-Erweiterung
- Embedding-Vektor-Spalte hinzufügen
- Vektor-Index für schnelle Suche
- Migration für bestehende Daten

### 3. Semantische Suche
- Cosine-Similarity-Berechnung
- Vektor-basierte Suche
- Hybrid-Suche (semantisch + Text)

### 4. Multimodal-Support
- Content-Type-Erkennung
- Bild-Embeddings
- Dokument-Embeddings
- Text-Embeddings

### 5. API-Erweiterung
- Embedding-Endpunkt
- Verbesserte Query-Ergebnisse
- Similarity-Scores
