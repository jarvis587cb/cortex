package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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
  query <text> [limit]       - Sucht nach Memories (Standard-Limit: 5)
  delete <id>                - Löscht ein Memory
  stats                     - Zeigt Statistiken
  help                      - Zeigt diese Hilfe

Umgebungsvariablen:
  CORTEX_API_URL   - API Base URL (Standard: %s)
  CORTEX_APP_ID    - App-ID (Standard: %s)
  CORTEX_USER_ID   - User-ID (Standard: %s)
  CORTEX_API_KEY   - Optional: API-Key für Auth (X-API-Key)

Flags (überschreiben Env):
  -url <url>    - API Base URL
  -app-id <id>  - App-ID
  -user-id <id> - User-ID

Beispiele:
  %s health
  %s store "Der Nutzer mag Kaffee"
  %s query "Kaffee" 10
  %s delete 1
  %s stats
`, prog, defaultBaseURL, defaultAppID, defaultUserID, prog, prog, prog, prog, prog)
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

func cmdQuery(client *cliClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Verwendung: query <text> [limit]")
	}
	query := args[0]
	limit := 5
	if len(args) >= 2 {
		var err error
		limit, err = strconv.Atoi(args[1])
		if err != nil || limit <= 0 {
			return fmt.Errorf("limit muss eine positive Ganzzahl sein")
		}
	}

	body := map[string]any{
		"appId":          client.appID,
		"externalUserId": client.userID,
		"query":          query,
		"limit":          limit,
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
