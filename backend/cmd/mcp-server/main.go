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
	"github.com/mikespook/unidb-mcp/internal/database"
	"github.com/mikespook/unidb-mcp/internal/handlers"
	"github.com/mikespook/unidb-mcp/internal/mcp"
	apprbac "github.com/mikespook/unidb-mcp/internal/rbac"
	"github.com/mikespook/unidb-mcp/internal/store"
)

func main() {
	// Command line flags
	addr := flag.String("addr", getEnv("ADDR", "localhost:9093"), "Server listen address")

	// Set default paths based on development mode
	defaultDataPath := "/app/data/config.db"
	defaultFrontendPath := "frontend/dist"
	if os.Getenv("DEV_MODE") == "true" {
		defaultDataPath = "data/config.db"
		defaultFrontendPath = "../frontend/dist"
	}
	dataPath := flag.String("data", getEnv("DATA_PATH", defaultDataPath), "SQLite database path")
	frontendPath := flag.String("frontend", getEnv("FRONTEND_PATH", defaultFrontendPath), "Frontend dist directory")
	flag.Parse()

	// Initialize SQLite store
	s, err := store.NewStore(*dataPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	// Initialize RBAC
	rbac := apprbac.New()

	// Initialize database connection manager
	manager := database.NewDriverManager()

	// Initialize bridge manager
	bridgeManager := handlers.NewBridgeManager(s)

	// Initialize handlers
	uiHandler := handlers.NewUIHandler(s, manager, *frontendPath)
	initHandler := handlers.NewInitHandler(s)
	userHandler := handlers.NewUserHandler(s, rbac)
	teamHandler := handlers.NewTeamHandler(s, rbac)
	mcpHandler := mcp.NewHandler(s, manager)
	bridgeHandler := handlers.NewBridgeHandler(bridgeManager)
	sseHandler := handlers.NewSSEHandler(bridgeManager)

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", uiHandler.Index)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(*frontendPath+"/assets"))))
	mux.HandleFunc("POST /login", uiHandler.Login)
	mux.HandleFunc("POST /logout", uiHandler.Logout)
	mux.HandleFunc("POST /init", initHandler.Setup)
	mux.HandleFunc("GET /api/ui/me", uiHandler.Me)
	mux.HandleFunc("GET /api/ui/init-status", initHandler.Status)

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

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
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

	// User management routes (session + RBAC protected inside handler)
	apiMux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.List(w, r)
		case http.MethodPost:
			userHandler.Create(w, r)
		case http.MethodPut:
			userHandler.Update(w, r)
		case http.MethodDelete:
			userHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("GET /users/{id}/jwt-secret", userHandler.GetJWTSecret)

	// Team management routes (session + RBAC protected inside handler)
	apiMux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			teamHandler.List(w, r)
		case http.MethodPost:
			teamHandler.Create(w, r)
		case http.MethodDelete:
			teamHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("GET /teams/{id}/users", teamHandler.GetUsers)
	apiMux.HandleFunc("POST /teams/{id}/users", teamHandler.AddUser)
	apiMux.HandleFunc("DELETE /teams/{id}/users", teamHandler.RemoveUser)
	apiMux.HandleFunc("GET /teams/{id}/dsns", teamHandler.GetDSNs)
	apiMux.HandleFunc("POST /teams/{id}/dsns", teamHandler.AddDSN)
	apiMux.HandleFunc("DELETE /teams/{id}/dsns", teamHandler.RemoveDSN)

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
		jwtMiddleware(s),
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

// jwtMiddleware validates Bearer tokens against all per-user JWT secrets.
func jwtMiddleware(s *store.Store) possum.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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

			secrets, err := s.ListAllJWTSecrets()
			if err != nil || len(secrets) == 0 {
				http.Error(w, `{"error": "Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			for _, secret := range secrets {
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(secret), nil
				})
				if err == nil && token.Valid {
					next(w, r)
					return
				}
			}

			http.Error(w, `{"error": "Invalid or expired token"}`, http.StatusUnauthorized)
		}
	}
}

// isPublicPath checks if the path should be accessible without JWT authentication
func isPublicPath(path string) bool {
	publicPaths := []string{"/", "/health", "/assets/", "/sse", "/mcp", "/ui/", "/ui/init-status", "/bridges/register"}
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
