// Command match starts the standalone matching service.
//
// This is an alternative deployment mode where matching runs in its own
// process. In the default (combined) mode, the matcher runs inside the
// signal server process. Set MATCH_STANDALONE=1 to run this separately.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/voicematch/voice-match/server/internal/match"
	"github.com/voicematch/voice-match/server/internal/store"
)

type config struct {
	HTTPAddr string        `yaml:"http_addr"`
	Redis    redisConfig   `yaml:"redis"`
}

type redisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func loadConfig(path string) (*config, error) {
	cfg := &config{
		HTTPAddr: ":8081",
		Redis:    redisConfig{Addr: "localhost:6379", DB: 0},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// noopSink is used when match runs standalone; match_found notifications
// are delivered via Redis pub/sub to the signal nodes instead.
type noopSink struct{}

func (n *noopSink) OnMatchFound(userA, userB, matchType string) {
	log.Printf("[match-standalone] match: %s <-> %s (%s)", userA, userB, matchType)
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config")
	flag.Parse()

	if v := os.Getenv("CONFIG_PATH"); v != "" {
		*configPath = v
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	redisClient := store.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	redisStore := store.NewRedisStore(redisClient)

	sink := &noopSink{}
	matcher := match.NewMatcher(redisStore, sink)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	matcher.Start(ctx)
	defer matcher.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "match",
			"uptime":  time.Since(startTime).Seconds(),
		})
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("match service listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-stop
	log.Println("match service shutting down...")

	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("match service stopped")
}

var startTime = time.Now()
