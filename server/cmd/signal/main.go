// Command signal starts the voice-match signaling server.
//
// It hosts:
//   - /ws        WebSocket upgrade endpoint
//   - /healthz   liveness probe
//   - /metrics   basic runtime metrics
//
// Early-stage deployment runs the Matcher in-process (combined mode).
// Set MATCH_STANDALONE=1 to disable the in-process matcher and connect
// to an external match service instead.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	ossignal "os/signal"
	"runtime"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/voicematch/voice-match/server/internal/match"
	"github.com/voicematch/voice-match/server/internal/signal"
	"github.com/voicematch/voice-match/server/internal/store"
)

// ---- Config ----

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Redis    RedisConfig    `yaml:"redis"`
	MySQL    MySQLConfig    `yaml:"mysql"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type MySQLConfig struct {
	DSN string `yaml:"dsn"`
}

func loadConfig(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{Addr: ":8080"},
		Redis:  RedisConfig{Addr: "localhost:6379", DB: 0},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Use defaults.
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// ---- stub implementations for injected interfaces ----

// stubUserValidator is a minimal UserValidator that always accepts.
// In production, replace with a real user-service client.
type stubUserValidator struct{}

func (s *stubUserValidator) ValidateToken(userID, token string) (bool, error) {
	if userID == "" || token == "" {
		return false, nil
	}
	return true, nil
}

func (s *stubUserValidator) GetUserProfile(userID string) (string, string, error) {
	return "user_" + userID, "https://example.com/avatar/" + userID + ".png", nil
}

// stubRTCService generates a fake RTC token and room name.
type stubRTCService struct{}

func (s *stubRTCService) CreateRoom(callID, callerID, calleeID string) (string, string, int64, error) {
	roomName := "room_" + callID
	token := "rtc_token_" + callID
	expiresAt := time.Now().Add(2 * time.Minute).Unix()
	return roomName, token, expiresAt, nil
}

// matchNotificationBridge connects the matcher's OnMatchFound callback
// to the signal handler's message delivery.
type matchNotificationBridge struct {
	hub *signal.Hub
}

func (b *matchNotificationBridge) OnMatchFound(userA, userB, matchType string) {
	// In production, this would create a CallSession and send match_found
	// + call_invite to both parties. For now, we send a match_found
	// notification to each user.
	log.Printf("[bridge] match found: %s <-> %s (%s)", userA, userB, matchType)
	// The actual call_invite flow is triggered when the caller initiates
	// via the RTC service. The signal handler's call_invite handler is
	// invoked from here in a full implementation.
}

// ---- main ----

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Override with env vars.
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		*configPath = v
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Initialize Redis.
	redisClient := store.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	redisStore := store.NewRedisStore(redisClient)

	// Initialize MySQL (optional in early stage).
	var mysqlStore *store.MySQLStore
	if cfg.MySQL.DSN != "" {
		mysqlStore, err = store.NewMySQLStore(cfg.MySQL.DSN)
		if err != nil {
			log.Printf("warn: mysql init failed: %v", err)
		} else {
			defer mysqlStore.Close()
			if err := mysqlStore.AutoMigrate(); err != nil {
				log.Printf("warn: mysql auto migrate: %v", err)
			}
			log.Println("mysql connected")
		}
	}

	// Initialize core components.
	hub := signal.NewHub()
	calls := signal.NewCallManager()
	hub.StartHeartbeatWatcher()

	users := &stubUserValidator{}
	rtc := &stubRTCService{}

	// Match service: in-process or standalone.
	var matchService signal.MatchService
	var matcher *match.Matcher

	if os.Getenv("MATCH_STANDALONE") != "1" {
		bridge := &matchNotificationBridge{hub: hub}
		matcher = match.NewMatcher(redisStore, bridge)
		matchService = matcher

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		matcher.Start(ctx)
		defer matcher.Stop()
		log.Println("in-process matcher started")
	} else {
		// In standalone mode, matchService would be an HTTP client.
		// For now, use a no-op.
		matchService = &noopMatchService{}
		log.Println("standalone match mode: matcher not started locally")
	}

	// Set up call ring-timeout callback.
	calls.SetOnRingTimeout(func(callID string) {
		// Notify both parties via hub.
		if _, ok := calls.Get(callID); !ok {
			return
		}
		// In production, send call_cancel to both.
		log.Printf("[main] ring timeout for call=%s", callID)
		calls.Remove(callID)
	})

	handler := signal.NewHandler(hub, calls, users, matchService, rtc)

	// HTTP routes.
	mux := http.NewServeMux()
	mux.Handle("/ws", handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]interface{}{
			"online":   hub.OnlineCount(),
			"calls":    calls.Count(),
			"goroutine": runtime.NumGoroutine(),
			"uptime":   time.Since(startTime).Seconds(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	ossignal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("signal server listening on %s", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("server stopped")
}

var startTime = time.Now()

// noopMatchService is used when match runs standalone.
type noopMatchService struct{}

func (n *noopMatchService) JoinPool(userID, matchType, city, geohash, destination string, tags []string, genderPreference string) (int, int, error) {
	return 0, 0, nil
}

func (n *noopMatchService) LeavePool(userID, matchType string) error {
	return nil
}
