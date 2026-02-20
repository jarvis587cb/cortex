package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cortex/internal/embeddings"
)

const defaultBaseURL = "http://localhost:9123"
const defaultAppID = "openclaw"
const defaultUserID = "default"

func main() {
	baseURL := envOr("CORTEX_API_URL", defaultBaseURL)
	appID := envOr("CORTEX_APP_ID", defaultAppID)
	userID := envOr("CORTEX_USER_ID", defaultUserID)
	apiKey := os.Getenv("CORTEX_API_KEY")

	fs := flag.NewFlagSet("global", flag.ExitOnError)
	fs.StringVar(&baseURL, "url", baseURL, "API base URL")
	fs.StringVar(&appID, "app-id", appID, "App ID")
	fs.StringVar(&userID, "user-id", userID, "User ID")
	_ = fs.Parse(os.Args[1:])

	args := fs.Args()
	if len(args) == 0 {
		printHelp(os.Args[0])
		os.Exit(0)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	client := &cliClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		appID:   appID,
		userID:  userID,
		apiKey:  apiKey,
	}

	var err error
	switch cmd {
	case "health":
		err = cmdHealth(client)
	case "store":
		err = cmdStore(client, cmdArgs)
	case "query":
		err = cmdQuery(client, cmdArgs)
	case "delete":
		err = cmdDelete(client, cmdArgs)
	case "stats":
		err = cmdStats(client)
	case "context-create":
		err = cmdContextCreate(client, cmdArgs)
	case "context-list":
		err = cmdContextList(client, cmdArgs)
	case "context-get":
		err = cmdContextGet(client, cmdArgs)
	case "generate-embeddings":
		err = cmdGenerateEmbeddings(client, cmdArgs)
	case "benchmark":
		err = cmdBenchmark(client, cmdArgs)
	case "benchmark-embeddings":
		err = cmdBenchmarkEmbeddings(cmdArgs)
	case "api-key":
		err = cmdAPIKey(cmdArgs)
	case "entity-add":
		err = cmdEntityAdd(client, cmdArgs)
	case "entity-get":
		err = cmdEntityGet(client, cmdArgs)
	case "relation-add":
		err = cmdRelationAdd(client, cmdArgs)
	case "relation-get":
		err = cmdRelationGet(client, cmdArgs)
	case "help", "-h", "--help":
		printHelp(os.Args[0])
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unbekannter Befehl: %s\n\n", cmd)
		printHelp(os.Args[0])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func printHelp(prog string) {
	fmt.Fprintf(os.Stderr, `Cortex CLI

Verwendung:
  %s <command> [args...]

Befehle:
  health                    - Prüft API-Status
  store <content> [metadata] - Speichert ein Memory (metadata optional JSON)
  query <text> [limit] [threshold] [seedIds] [metadataFilter] - Suche (limit=5, threshold=0.2, seedIds z.B. 1,2,3, metadataFilter z.B. '{"typ":"persönlich"}')
  delete <id>                - Löscht ein Memory
  stats                     - Zeigt Statistiken
  entity-add <entity> <key> <value> - Fact zu einer Entity hinzufügen
  entity-get <entity>      - Entity mit allen Fakten abrufen
  relation-add <from> <to> <type> - Relation zwischen Entities anlegen
  relation-get <from>      - Alle Relations von einer Entity abrufen
  context-create <agentId> [memoryType] [payload] - Agent-Context anlegen (memoryType: episodic|semantic|procedural|working)
  context-list [agentId]    - Agent-Contexts auflisten
  context-get <id>          - Ein Agent-Context abrufen
  generate-embeddings [batchSize] - Embeddings für Memories nachziehen (Standard: 10, Max: 100)
  benchmark [count]         - Performance-Benchmark (Standard: 20 Requests)
  benchmark-embeddings [count] [service] - Benchmark Embedding-Generierung (count=50, service=local|gte|both)
  api-key <create|delete|show> [env_file] - API-Key verwalten (Standard: .env im Projekt)
  help                      - Zeigt diese Hilfe

Umgebungsvariablen:
  CORTEX_API_URL   - API Base URL (Standard: %s)
  CORTEX_APP_ID    - App-ID (Standard: %s)
  CORTEX_USER_ID   - User-ID (Standard: %s)
  CORTEX_API_KEY   - Optional: API-Key für Auth (nur für Produktion; lokale Installation benötigt keinen)

Flags (überschreiben Env):
  -url <url>    - API Base URL
  -app-id <id>  - App-ID
  -user-id <id> - User-ID

Beispiele:
  %s health
  %s store "Der Nutzer mag Kaffee"
  %s query "Kaffee" 10 0.2
  %s query "Kaffee" 10 0.5 "1,2,3"
  %s query "Kaffee" 10 0.5 "" '{"typ":"persönlich"}'
  %s delete 1
  %s stats
  %s entity-add carsten lieblingsfarbe blau
  %s entity-get carsten
  %s relation-add carsten typescript programmiert
  %s relation-get carsten
  %s context-create "my-agent" episodic '{}'
  %s context-list "my-agent"
  %s context-get 1
  %s generate-embeddings 100
  %s benchmark 50
  %s benchmark-embeddings 100 local
  %s api-key create
  %s api-key show
`, prog, defaultBaseURL, defaultAppID, defaultUserID, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog, prog)
}

type cliClient struct {
	baseURL string
	appID   string
	userID  string
	apiKey  string
}

func (c *cliClient) do(method, path string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func cmdHealth(client *cliClient) error {
	data, code, err := client.do(http.MethodGet, "/health", nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("API nicht erreichbar (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	fmt.Println("API ist erreichbar")
	return nil
}

func cmdStore(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: store <content> [metadata]")
	}
	content := args[0]
	var metadata map[string]any
	if len(args) >= 2 {
		if err := json.Unmarshal([]byte(args[1]), &metadata); err != nil {
			return fmt.Errorf("metadata muss gültiges JSON sein: %w", err)
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}

	body := map[string]any{
		"appId":          client.appID,
		"externalUserId": client.userID,
		"content":        content,
		"metadata":      metadata,
	}
	data, code, err := client.do(http.MethodPost, "/seeds", body)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Speichern (HTTP %d): %s", code, string(data))
	}
	var res struct {
		ID      int64  `json:"id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &res); err == nil && res.ID != 0 {
		fmt.Printf("Memory gespeichert (ID: %d)\n", res.ID)
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func parseSeedIDs(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("ungültige seedIds: %q (erwarte kommagetrennte positive Zahlen)", s)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func cmdQuery(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: query <text> [limit] [threshold] [seedIds] [metadataFilter]")
	}
	query := args[0]
	limit := 5
	threshold := 0.2
	var seedIDs []int64
	var metadataFilter map[string]any
	if len(args) >= 2 {
		var err error
		limit, err = strconv.Atoi(args[1])
		if err != nil || limit <= 0 {
			return fmt.Errorf("limit muss eine positive Ganzzahl sein")
		}
	}
	if len(args) >= 3 {
		var err error
		threshold, err = strconv.ParseFloat(args[2], 64)
		if err != nil || threshold < 0 || threshold > 1 {
			return fmt.Errorf("threshold muss eine Zahl zwischen 0 und 1 sein")
		}
	}
	// Parse optional arguments: seedIDs and/or metadataFilter
	// If args[3] looks like JSON (starts with '{'), treat it as metadataFilter
	// Otherwise, treat it as seedIDs
	if len(args) >= 4 && args[3] != "" {
		trimmed := strings.TrimSpace(args[3])
		// Check if args[3] looks like JSON (metadataFilter)
		if len(trimmed) > 0 && trimmed[0] == '{' {
			if err := json.Unmarshal([]byte(args[3]), &metadataFilter); err != nil {
				return fmt.Errorf("metadataFilter muss gültiges JSON sein: %w", err)
			}
		} else {
			// Treat as seedIDs
			var err error
			seedIDs, err = parseSeedIDs(args[3])
			if err != nil {
				return err
			}
		}
	}
	// If args[4] exists, it's metadataFilter (either because args[3] was seedIDs, or args[3] was empty)
	if len(args) >= 5 && args[4] != "" {
		if err := json.Unmarshal([]byte(args[4]), &metadataFilter); err != nil {
			return fmt.Errorf("metadataFilter muss gültiges JSON sein: %w", err)
		}
	}

	body := map[string]any{
		"appId":          client.appID,
		"externalUserId": client.userID,
		"query":          query,
		"limit":          limit,
		"threshold":      threshold,
	}
	if len(seedIDs) > 0 {
		body["seedIds"] = seedIDs
	}
	if len(metadataFilter) > 0 {
		body["metadataFilter"] = metadataFilter
	}
	data, code, err := client.do(http.MethodPost, "/seeds/query", body)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler bei der Suche (HTTP %d): %s", code, string(data))
	}

	var results []map[string]any
	if err := json.Unmarshal(data, &results); err != nil {
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(string(data))
	fmt.Printf("Gefunden: %d Memories\n", len(results))
	return nil
}

func cmdDelete(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: delete <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || id <= 0 {
		return fmt.Errorf("id muss eine positive Ganzzahl sein")
	}

	path := "/seeds/" + strconv.FormatInt(id, 10) + "?appId=" + url.QueryEscape(client.appID) + "&externalUserId=" + url.QueryEscape(client.userID)
	data, code, err := client.do(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if code == http.StatusNotFound {
		return fmt.Errorf("Memory nicht gefunden (ID: %d)", id)
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Löschen (HTTP %d): %s", code, string(data))
	}
	fmt.Println("Memory gelöscht")
	return nil
}

func cmdStats(client *cliClient) error {
	data, code, err := client.do(http.MethodGet, "/stats", nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Laden der Statistiken (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	return nil
}

func cmdContextCreate(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: context-create <agentId> [memoryType] [payload]")
	}
	agentID := args[0]
	memoryType := "episodic"
	if len(args) >= 2 {
		memoryType = strings.ToLower(strings.TrimSpace(args[1]))
	}
	payload := map[string]any{}
	if len(args) >= 3 && args[2] != "" && args[2] != "{}" {
		if err := json.Unmarshal([]byte(args[2]), &payload); err != nil {
			return fmt.Errorf("payload muss gültiges JSON sein: %w", err)
		}
	}
	body := map[string]any{
		"appId":          client.appID,
		"externalUserId": client.userID,
		"agentId":        agentID,
		"memoryType":    memoryType,
		"payload":       payload,
	}
	data, code, err := client.do(http.MethodPost, "/agent-contexts", body)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Anlegen (HTTP %d): %s", code, string(data))
	}
	var res struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &res); err == nil && res.ID != 0 {
		fmt.Printf("Agent-Context erstellt (ID: %d)\n", res.ID)
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func cmdContextList(client *cliClient, args []string) error {
	path := "/agent-contexts?appId=" + url.QueryEscape(client.appID) + "&externalUserId=" + url.QueryEscape(client.userID)
	if len(args) >= 1 && args[0] != "" {
		path += "&agentId=" + url.QueryEscape(args[0])
	}
	data, code, err := client.do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Auflisten (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	return nil
}

func cmdContextGet(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: context-get <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || id <= 0 {
		return fmt.Errorf("id muss eine positive Ganzzahl sein")
	}
	path := "/agent-contexts/" + strconv.FormatInt(id, 10) + "?appId=" + url.QueryEscape(client.appID) + "&externalUserId=" + url.QueryEscape(client.userID)
	data, code, err := client.do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if code == http.StatusNotFound {
		return fmt.Errorf("Agent-Context nicht gefunden (ID: %d)", id)
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	return nil
}

func cmdGenerateEmbeddings(client *cliClient, args []string) error {
	batchSize := 10
	if len(args) >= 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 || n > 100 {
			return fmt.Errorf("batchSize muss zwischen 1 und 100 liegen")
		}
		batchSize = n
	}
	path := "/seeds/generate-embeddings?batchSize=" + strconv.Itoa(batchSize)
	data, code, err := client.do(http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	return nil
}

func cmdBenchmark(client *cliClient, args []string) error {
	count := 20
	if len(args) >= 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return fmt.Errorf("count muss eine positive Ganzzahl sein")
		}
		count = n
	}

	fmt.Printf("Starte Benchmark (N=%d, API=%s)...\n", count, client.baseURL)

	var healthTimes, storeTimes, queryTimes, deleteTimes []float64
	var storeIDs []int64

	// Benchmark health
	for i := 0; i < count; i++ {
		start := time.Now()
		_, code, err := client.do(http.MethodGet, "/health", nil)
		if err != nil || code != http.StatusOK {
			return fmt.Errorf("health check fehlgeschlagen: %v", err)
		}
		healthTimes = append(healthTimes, time.Since(start).Seconds())
	}

	// Benchmark store, query, delete
	for i := 0; i < count; i++ {
		content := fmt.Sprintf("benchmark-memory-%d-%d", i, time.Now().UnixNano())
		
		// Store
		start := time.Now()
		body := map[string]any{
			"appId":          client.appID,
			"externalUserId": client.userID,
			"content":        content,
			"metadata":       map[string]any{"type": "benchmark"},
		}
		data, code, err := client.do(http.MethodPost, "/seeds", body)
		if err != nil || code != http.StatusOK {
			return fmt.Errorf("store fehlgeschlagen: %v", err)
		}
		storeTimes = append(storeTimes, time.Since(start).Seconds())
		
		var res struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(data, &res); err == nil && res.ID != 0 {
			storeIDs = append(storeIDs, res.ID)
		}

		// Query
		start = time.Now()
		queryBody := map[string]any{
			"appId":          client.appID,
			"externalUserId": client.userID,
			"query":          fmt.Sprintf("benchmark-memory-%d", i),
			"limit":          5,
		}
		_, code, err = client.do(http.MethodPost, "/seeds/query", queryBody)
		if err != nil || code != http.StatusOK {
			return fmt.Errorf("query fehlgeschlagen: %v", err)
		}
		queryTimes = append(queryTimes, time.Since(start).Seconds())

		// Delete
		if i < len(storeIDs) {
			start = time.Now()
			path := fmt.Sprintf("/seeds/%d?appId=%s&externalUserId=%s", storeIDs[i], url.QueryEscape(client.appID), url.QueryEscape(client.userID))
			_, code, err = client.do(http.MethodDelete, path, nil)
			if err != nil || code != http.StatusOK {
				return fmt.Errorf("delete fehlgeschlagen: %v", err)
			}
			deleteTimes = append(deleteTimes, time.Since(start).Seconds())
		}
	}

	// Berechne Statistiken
	summarize := func(name string, times []float64) {
		if len(times) == 0 {
			return
		}
		var sum, min, max float64
		min = times[0]
		max = times[0]
		for _, t := range times {
			sum += t
			if t < min {
				min = t
			}
			if t > max {
				max = t
			}
		}
		avg := sum / float64(len(times))
		fmt.Printf("%s n=%d avg=%.4fs min=%.4fs max=%.4fs\n", name, len(times), avg, min, max)
	}

	fmt.Println("\nBenchmark Ergebnisse (N=" + strconv.Itoa(count) + ", API=" + client.baseURL + ")")
	fmt.Println("==========================================")
	summarize("health", healthTimes)
	summarize("store", storeTimes)
	summarize("query", queryTimes)
	summarize("delete", deleteTimes)

	return nil
}

func cmdBenchmarkEmbeddings(args []string) error {
	count := 50
	serviceType := "both" // local, gte, both
	
	if len(args) >= 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return fmt.Errorf("count muss eine positive Ganzzahl sein")
		}
		count = n
	}
	if len(args) >= 2 {
		serviceType = strings.ToLower(args[1])
		if serviceType != "local" && serviceType != "gte" && serviceType != "both" {
			return fmt.Errorf("service muss 'local', 'gte' oder 'both' sein")
		}
	}

	fmt.Printf("Starte Embedding-Benchmark (N=%d, Service=%s)...\n\n", count, serviceType)

	// Test-Texte mit verschiedenen Längen
	testTexts := []struct {
		name string
		text string
	}{
		{"Kurz (10 Wörter)", "Dies ist ein kurzer Testtext mit genau zehn Wörtern für den Benchmark"},
		{"Mittel (50 Wörter)", strings.Repeat("Dies ist ein Testtext mit mehreren Wörtern. ", 10)},
		{"Lang (200 Wörter)", strings.Repeat("Dies ist ein längerer Testtext für umfangreichere Benchmarks. ", 40)},
	}

	// Benchmark-Funktion
	benchmarkService := func(name string, service embeddings.EmbeddingService) {
		fmt.Printf("=== %s ===\n", name)
		
		for _, testCase := range testTexts {
			// Single Embedding Benchmark
			var singleTimes []float64
			for i := 0; i < count; i++ {
				start := time.Now()
				_, err := service.GenerateEmbedding(testCase.text, "text/plain")
				if err != nil {
					fmt.Printf("  Fehler bei %s: %v\n", testCase.name, err)
					continue
				}
				singleTimes = append(singleTimes, time.Since(start).Seconds())
			}
			
			// Batch Benchmark (10 Texte auf einmal)
			var batchTimes []float64
			batchTexts := make([]string, 10)
			for i := range batchTexts {
				batchTexts[i] = testCase.text
			}
			for i := 0; i < count/10; i++ {
				start := time.Now()
				_, err := service.GenerateEmbeddingsBatch(batchTexts, "text/plain")
				if err != nil {
					fmt.Printf("  Fehler bei Batch %s: %v\n", testCase.name, err)
					continue
				}
				batchTimes = append(batchTimes, time.Since(start).Seconds())
			}
			
			// Statistiken berechnen
			summarize := func(times []float64) (avg, min, max float64) {
				if len(times) == 0 {
					return 0, 0, 0
				}
				var sum float64
				min = times[0]
				max = times[0]
				for _, t := range times {
					sum += t
					if t < min {
						min = t
					}
					if t > max {
						max = t
					}
				}
				avg = sum / float64(len(times))
				return avg, min, max
			}
			
			singleAvg, singleMin, singleMax := summarize(singleTimes)
			batchAvg, batchMin, batchMax := summarize(batchTimes)
			
			fmt.Printf("  %s:\n", testCase.name)
			fmt.Printf("    Single: n=%d avg=%.4fms min=%.4fms max=%.4fms\n", 
				len(singleTimes), singleAvg*1000, singleMin*1000, singleMax*1000)
			if len(batchTimes) > 0 {
				fmt.Printf("    Batch (10): n=%d avg=%.4fms min=%.4fms max=%.4fms (pro Text: %.4fms)\n", 
					len(batchTimes), batchAvg*1000, batchMin*1000, batchMax*1000, (batchAvg/10)*1000)
			}
		}
		fmt.Println()
	}

	// Local Hash-based Service
	if serviceType == "local" || serviceType == "both" {
		localService := embeddings.NewLocalEmbeddingService()
		benchmarkService("Local Hash-based Service", localService)
	}

	// GTE Service (falls verfügbar)
	if serviceType == "gte" || serviceType == "both" {
		modelPath := os.Getenv("CORTEX_EMBEDDING_MODEL_PATH")
		if modelPath == "" {
			homeDir, _ := os.UserHomeDir()
			modelPath = filepath.Join(homeDir, ".openclaw", "gte-small.gtemodel")
		}
		
		// Expandiere ~ zu Home-Verzeichnis
		if strings.HasPrefix(modelPath, "~") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				modelPath = filepath.Join(homeDir, strings.TrimPrefix(modelPath, "~"))
			}
		}
		
		if _, err := os.Stat(modelPath); err == nil {
			gteService, err := embeddings.NewGTEEmbeddingService(modelPath)
			if err != nil {
				fmt.Printf("Warnung: GTE-Service konnte nicht geladen werden: %v\n", err)
				fmt.Printf("Modell-Pfad: %s\n\n", modelPath)
			} else {
				defer gteService.Close()
				benchmarkService("GTE-Small Service", gteService)
			}
		} else {
			fmt.Printf("Warnung: GTE-Modell nicht gefunden: %s\n", modelPath)
			fmt.Printf("Setze CORTEX_EMBEDDING_MODEL_PATH oder verwende 'local' für Hash-Service\n\n")
		}
	}

	return nil
}

func cmdEntityAdd(client *cliClient, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("Verwendung: entity-add <entity> <key> <value>")
	}
	entity := args[0]
	key := args[1]
	value := args[2]

	// Versuche value als JSON zu parsen, falls es JSON ist
	var valueAny any = value
	if len(value) > 0 && (value[0] == '{' || value[0] == '[') {
		if err := json.Unmarshal([]byte(value), &valueAny); err == nil {
			// Erfolgreich als JSON geparst
		} else {
			// Nicht JSON, als String verwenden
			valueAny = value
		}
	}

	body := map[string]any{
		"key":   key,
		"value": valueAny,
	}
	path := "/entities?entity=" + url.QueryEscape(entity)
	data, code, err := client.do(http.MethodPost, path, body)
	if err != nil {
		return err
	}
	if code != http.StatusNoContent {
		return fmt.Errorf("Fehler beim Hinzufügen des Facts (HTTP %d): %s", code, string(data))
	}
	fmt.Printf("Fact '%s' zu Entity '%s' hinzugefügt\n", key, entity)
	return nil
}

func cmdEntityGet(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: entity-get <entity>")
	}
	entity := args[0]

	path := "/entities?name=" + url.QueryEscape(entity)
	data, code, err := client.do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if code == http.StatusNotFound {
		return fmt.Errorf("Entity nicht gefunden: %s", entity)
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Abrufen der Entity (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	return nil
}

func cmdRelationAdd(client *cliClient, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("Verwendung: relation-add <from> <to> <type>")
	}
	from := args[0]
	to := args[1]
	relType := args[2]

	body := map[string]any{
		"from": from,
		"to":   to,
		"type": relType,
	}
	data, code, err := client.do(http.MethodPost, "/relations", body)
	if err != nil {
		return err
	}
	if code != http.StatusNoContent {
		return fmt.Errorf("Fehler beim Anlegen der Relation (HTTP %d): %s", code, string(data))
	}
	fmt.Printf("Relation '%s' von '%s' zu '%s' angelegt\n", relType, from, to)
	return nil
}

func cmdRelationGet(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: relation-get <from>")
	}
	from := args[0]

	path := "/relations?entity=" + url.QueryEscape(from)
	data, code, err := client.do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("Fehler beim Abrufen der Relations (HTTP %d): %s", code, string(data))
	}
	fmt.Println(string(data))
	return nil
}

func cmdAPIKey(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: api-key <create|delete|show> [env_file]")
	}

	cmd := args[0]
	envFile := ""
	if len(args) >= 2 {
		envFile = args[1]
	} else {
		// Suche .env im aktuellen Verzeichnis oder Projekt-Root
		if _, err := os.Stat(".env"); err == nil {
			envFile = ".env"
		} else {
			// Versuche Projekt-Root zu finden
			cwd, _ := os.Getwd()
			projectRoot := cwd
			for {
				if _, err := os.Stat(filepath.Join(projectRoot, ".env")); err == nil {
					envFile = filepath.Join(projectRoot, ".env")
					break
				}
				parent := filepath.Dir(projectRoot)
				if parent == projectRoot {
					envFile = ".env"
					break
				}
				projectRoot = parent
			}
		}
	}

	const varName = "CORTEX_API_KEY"
	const keyPrefix = "ck_"

	switch cmd {
	case "create":
		// Generiere Key
		keyBytes := make([]byte, 32)
		if _, err := rand.Read(keyBytes); err != nil {
			return fmt.Errorf("Fehler beim Generieren des Keys: %w", err)
		}
		key := keyPrefix + hex.EncodeToString(keyBytes)

		// Lese bestehende .env
		var lines []string
		if data, err := os.ReadFile(envFile); err == nil {
			lines = strings.Split(string(data), "\n")
		}

		// Entferne alte CORTEX_API_KEY Zeile
		var newLines []string
		for _, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), varName+"=") {
				newLines = append(newLines, line)
			}
		}

		// Füge neue Zeile hinzu
		newLines = append(newLines, varName+"="+key)

		// Schreibe .env
		if err := os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")+"\n"), 0644); err != nil {
			return fmt.Errorf("Fehler beim Schreiben von %s: %w", envFile, err)
		}

		fmt.Printf("✓ API-Key in %s gesetzt\n", envFile)
		fmt.Printf("\nNeuer API-Key (einmalig sichtbar – sicher aufbewahren):\n")
		fmt.Printf("  %s\n\n", key)
		fmt.Printf("Server starten mit: export %s=%s; ./cortex-server\n", varName, key)

	case "delete":
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			fmt.Printf("Datei %s nicht gefunden – nichts zu löschen.\n", envFile)
			return nil
		}

		data, err := os.ReadFile(envFile)
		if err != nil {
			return fmt.Errorf("Fehler beim Lesen von %s: %w", envFile, err)
		}

		lines := strings.Split(string(data), "\n")
		var newLines []string
		found := false
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), varName+"=") {
				found = true
				continue
			}
			newLines = append(newLines, line)
		}

		if !found {
			fmt.Printf("Kein Eintrag %s in %s gefunden.\n", varName, envFile)
			return nil
		}

		if err := os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			return fmt.Errorf("Fehler beim Schreiben von %s: %w", envFile, err)
		}

		fmt.Printf("✓ API-Key aus %s entfernt\n", envFile)

	case "show":
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			fmt.Printf("Datei %s nicht gefunden.\n", envFile)
			return nil
		}

		data, err := os.ReadFile(envFile)
		if err != nil {
			return fmt.Errorf("Fehler beim Lesen von %s: %w", envFile, err)
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, varName+"=") {
				key := strings.TrimPrefix(line, varName+"=")
				key = strings.Trim(key, `"'`)
				if len(key) > 4 {
					fmt.Printf("Aktueller Key in %s (letzte 4 Zeichen): ...%s\n", envFile, key[len(key)-4:])
				} else {
					fmt.Printf("Key in %s gesetzt (Länge %d).\n", envFile, len(key))
				}
				return nil
			}
		}

		fmt.Printf("Kein %s in %s gesetzt.\n", varName, envFile)

	default:
		return fmt.Errorf("Unbekannter Befehl: %s. Verwende: create, delete oder show", cmd)
	}

	return nil
}
