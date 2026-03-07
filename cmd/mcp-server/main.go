package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mikespook/possum"
	"github.com/mikespook/unidb-mcp/internal/config"
	"github.com/mikespook/unidb-mcp/internal/database"
	"github.com/mikespook/unidb-mcp/internal/handlers"
	"github.com/mikespook/unidb-mcp/internal/mcp"
	"github.com/mikespook/unidb-mcp/internal/store"
)

func main() {
	// Command line flags
	addr := flag.String("addr", getEnv("ADDR", "localhost:9093"), "Server listen address")

	// Set default data path based on development mode
	defaultDataPath := "/app/data/config.db"
	if os.Getenv("DEV_MODE") == "true" {
		defaultDataPath = "data/config.db"
	}
	dataPath := flag.String("data", getEnv("DATA_PATH", defaultDataPath), "SQLite database path")
	resetUIPassword := flag.Bool("reset-ui-password", false, "Reset the UI password and exit")
	flag.Parse()

	// Initialize SQLite store
	s, err := store.NewStore(*dataPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	// Handle -reset-ui-password flag
	if *resetUIPassword {
		newPass, err := config.ResetUIPassword(s)
		if err != nil {
			log.Fatalf("Failed to reset UI password: %v", err)
		}
		fmt.Printf("UI password reset. New password: %s\n", newPass)
		return
	}

	// Load JWT configuration (auto-generates and persists secret if not set)
	jwtCfg, err := config.LoadJWTConfig(s)
	if err != nil {
		log.Fatalf("Failed to load JWT config: %v", err)
	}

	// Load UI password configuration
	uiPassCfg, err := config.LoadUIPassword(s)
	if err != nil {
		log.Fatalf("Failed to load UI password config: %v", err)
	}

	// Initialize database connection manager
	manager := database.NewDriverManager()

	// Initialize bridge manager
	bridgeManager := handlers.NewBridgeManager(s)

	// Initialize handlers
	uiHandler := handlers.NewUIHandler(s, manager, uiPassCfg)
	mcpHandler := mcp.NewHandler(s, manager)
	bridgeHandler := handlers.NewBridgeHandler(bridgeManager)
	sseHandler := handlers.NewSSEHandler(bridgeManager)

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", uiHandler.Index)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/app.js", uiHandler.AppJS)
	mux.HandleFunc("POST /login", uiHandler.Login)
	mux.HandleFunc("POST /logout", uiHandler.Logout)
	mux.HandleFunc("GET /api/ui/me", uiHandler.Me)

	// Bridge SSE endpoint (public, authenticated via query params)
	mux.HandleFunc("/sse", sseHandler.ServeHTTP)

	// API routes
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/dsns", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			uiHandler.ListDSNs(w, r)
		case http.MethodPost:
			uiHandler.CreateDSN(w, r)
		case http.MethodPut:
			uiHandler.UpdateDSN(w, r)
		case http.MethodDelete:
			uiHandler.DeleteDSN(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("POST /dsns/{id}/test", uiHandler.TestDSN)
	apiMux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("MCP %s from %s | Accept: %s", r.Method, r.RemoteAddr, r.Header.Get("Accept"))

		// Handle GET for SSE stream (Streamable HTTP transport)
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			// Keep connection open until client disconnects
			<-r.Context().Done()
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req mcp.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			resp := mcp.Response{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &mcp.Error{
					Code:    mcp.ParseError,
					Message: "Invalid JSON-RPC request",
					Data:    err.Error(),
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		log.Printf("MCP method: %s", req.Method)
		resp := mcpHandler.HandleRequest(req)
		if resp.IsNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// UI auth routes (session-protected)
	apiMux.HandleFunc("POST /ui/password", uiHandler.SessionMiddleware(uiHandler.ChangePassword))

	// Bridge API routes
	apiMux.HandleFunc("/bridges/register", bridgeHandler.Register)
	apiMux.HandleFunc("/bridges", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			bridgeHandler.List(w, r)
		case http.MethodPut:
			bridgeHandler.Update(w, r)
		case http.MethodDelete:
			bridgeHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Apply middleware chain to API routes
	apiHandler := possum.Chain(
		apiMux.ServeHTTP,
		possum.Log,
		possum.Cors(nil),
		jwtMiddleware(jwtCfg),
	)

	// Mount API routes
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))

	// Start server
	fmt.Printf("🚀 UniDB MCP Server starting on http://%s\n", *addr)
	fmt.Printf("📊 Web UI: http://%s/\n", *addr)
	fmt.Printf("🔌 MCP Endpoint: http://%s/mcp\n", *addr)

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// jwtMiddleware creates JWT authentication middleware
func jwtMiddleware(cfg *config.JWTConfig) possum.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for certain paths
			if isPublicPath(r.URL.Path) {
				next(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "Missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error": "Invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.Secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error": "Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			next(w, r)
		}
	}
}

// isPublicPath checks if the path should be accessible without authentication
func isPublicPath(path string) bool {
	publicPaths := []string{"/", "/health", "/app.js", "/sse", "/mcp", "/ui/", "/bridges/register"}
	for _, p := range publicPaths {
		if path == p || strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// getEnv gets environment variable with default fallback
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
