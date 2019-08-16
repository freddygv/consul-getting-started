package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/matryer/way"
	"golang.org/x/time/rate"
)

const (
	limiterRate  = 0.1
	limiterBurst = 2
)

func main() {
	var (
		httpAddr   = flag.String("addr", "localhost:8080", "Hello service address.")
		configFile = flag.String("cfg-file", "config.json", "Path to config file.")
	)
	flag.Parse()

	log.Printf("[INFO] Starting server...")
	log.Printf("[INFO] Hello service with TTL check listening on %s", *httpAddr)

	s := newServer(*configFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("[INFO] Running TTL check keep-alive")
	s.runTTL(ctx)

	for _, key := range SliceVal(s.cfg.ToWatch) {
		log.Printf("[INFO] Running watch for key '%s'", key)
		s.watchKV(ctx, key)
	}

	log.Fatal(http.ListenAndServe(*httpAddr, s.router))
}

type server struct {
	router *way.Router
	cfg    *serverConfig
}

func newServer(cfgFile string) *server {
	config, err := loadConfig(cfgFile)
	if err != nil {
		log.Printf("[ERR] failed to load config from file '%s', using default. err: %v", cfgFile, err)
	}
	config = config.finalize()

	s := server{
		router: way.NewRouter(),
		cfg:    config,
	}

	s.router.HandleFunc("GET", "/hello", s.handleHello())
	s.router.HandleFunc("PUT", "/health/pass", s.enableHealth())
	s.router.HandleFunc("PUT", "/health/fail", s.disableHealth())

	return &s
}

func (s *server) handleHello() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.cfg.mu.RLock()
		defer s.cfg.mu.RUnlock()

		switch StringVal(s.cfg.Language) {
		case "french":
			fmt.Fprintln(w, "Bonjour Monde")
		case "portuguese":
			fmt.Fprintln(w, "Olá Mundo")
		case "spanish":
			fmt.Fprintln(w, "Hola Mundo")
		default:
			fmt.Fprintln(w, "Hello World")
		}
	}
}

func (s *server) disableHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.cfg.mu.Lock()
		defer s.cfg.mu.Unlock()

		s.cfg.EnableChecks = BoolPtr(false)
		fmt.Fprintln(w, "Health endpoint disabled.")
	}
}

func (s *server) enableHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.cfg.mu.Lock()
		defer s.cfg.mu.Unlock()

		s.cfg.EnableChecks = BoolPtr(true)
		fmt.Fprintln(w, "Health endpoint enabled.")
	}
}

func (s *server) runTTL(ctx context.Context) {
	ticker := time.NewTicker(DurationVal(s.cfg.TTLInterval))
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			default:
				s.cfg.mu.RLock()
				{
					if BoolVal(s.cfg.EnableChecks) {
						target := StringVal(s.cfg.ConsulAddr) + StringVal(s.cfg.TTLEndpoint) + StringVal(s.cfg.TTLID)
						_, err := http.NewRequest("PUT", target, nil)
						if err != nil {
							log.Printf("[ERR] failed to update TTL check: %v", err)
						}
					}
				}
				s.cfg.mu.RUnlock()

				<-ticker.C
			}
		}
	}()
}

// watchKV watches a Key/Value pair in Consul for changes and sets the value internally
// See below for implementation details:
// https://www.consul.io/api/features/blocking.html#implementation-details
func (s *server) watchKV(ctx context.Context, key string) {
	var index uint64 = 1
	var lastIndex uint64

	limiter := rate.NewLimiter(limiterRate, limiterBurst)

	for {
		// Wait until limiter allows request to happen
		if err := limiter.Wait(nil); err != nil {
			log.Println("failed to wait for limiter")
			continue
		}

		// Make blocking query to watch key
		target := fmt.Sprintf("%s%s%s?index=%d", StringVal(s.cfg.ConsulAddr), StringVal(s.cfg.KVPath), key, index)

		resp, err := http.Get(target)
		if err != nil {
			log.Printf("failed to get '%s': %v", target, err)
			continue
		}
		defer resp.Body.Close()

		// Parse the raft index for this key (X-Consul-Index)
		header := resp.Header
		indexStr := header.Get("X-Consul-Index")
		if indexStr != "" {
			index, err = strconv.ParseUint(indexStr, 10, 64)
			if err != nil {
				log.Printf("failed to parse X-Consul-Index: %v", err)
				continue
			}
		}
		// Reset if it goes backwards or is 0
		// See: https://www.consul.io/api/features/blocking.html#implementation-details
		if index < lastIndex || index == 0 {
			index = 1
			lastIndex = 1

			// TODO: Continuing implies we don't trust the data on the server
			continue
		}

		data := make([]keyResponse, 0)
		json.NewDecoder(resp.Body).Decode(&data)

		// We are not recursing on a key-prefix so these arrays will only return one value
		decoded, err := base64.StdEncoding.DecodeString(data[0].Value)
		if err != nil {
			log.Printf("failed to decode value: '%s'", data[0].Value)
			continue
		}

		s.cfg.mu.Lock()
		{
			// Set new value, may be idempotent
			switch key {
			case "language":
				s.cfg.Language = StringPtr(string(decoded))
			}
		}
		s.cfg.mu.Unlock()

		log.Printf("[INFO] %s updated: %s", key, decoded)

		lastIndex = index
	}
}

type keyResponse struct {
	LockIndex   uint64
	Key         string
	Flags       int
	Value       string
	CreateIndex uint64
	ModifyIndex uint64
}
