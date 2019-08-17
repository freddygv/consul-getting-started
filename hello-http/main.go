package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/matryer/way"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

	s := newServer(*configFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, key := range SliceVal(s.cfg.ToWatch) {
		log.Printf("[INFO] Running watch for key '%s'", key)
		go s.watchKV(ctx, key)
	}

	go s.captureReload(ctx, *configFile)

	log.Printf("[INFO] Hello service with HTTP check listening on %s", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, s.router))
}

type server struct {
	router *way.Router
	cfg    *serverConfig
}

func newServer(cfgFile string) *server {
	config, err := loadConfig(cfgFile)
	if err != nil {
		log.Printf("[WARN] failed to load config from file '%s', using default. err: %v", cfgFile, err)
	}
	config = config.merge(defaultConfig())

	s := server{
		router: way.NewRouter(),
		cfg:    config,
	}

	s.router.HandleFunc("GET", "/hello", s.handleHello())
	s.router.HandleFunc("GET", "/healthz", s.handleHealth())
	s.router.HandleFunc("PUT", "/health/pass", s.enableHealth())
	s.router.HandleFunc("PUT", "/health/fail", s.disableHealth())

	return &s
}

// Reload config from file on HUP
func (s *server) captureReload(ctx context.Context, cfgFile string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	for {
		select {
		case sig := <-sigCh:
			log.Printf("[INFO] captured signal: %v. reloading config...", sig)
			config, err := loadConfig(cfgFile)
			if err != nil {
				log.Printf("[WARN] failed to load config from file '%s', using default. err: %v", cfgFile, err)
			}
			s.cfg.mu.Lock()
			{
				s.cfg = config.merge(s.cfg)
			}
			s.cfg.mu.Unlock()
		}
	}
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

func (s *server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.cfg.mu.RLock()
		defer s.cfg.mu.RUnlock()

		// Fail check if checks aren't enabled
		if !BoolVal(s.cfg.EnableChecks) {
			w.WriteHeader(http.StatusGone)
			return
		}

		fmt.Fprintln(w, "I'm alive")
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

// watchKV watches a Key/Value pair in Consul for changes and sets the value internally
// See below for implementation details:
// https://www.consul.io/api/features/blocking.html#implementation-details
func (s *server) watchKV(ctx context.Context, key string) {
	var index uint64 = 1
	var lastIndex uint64

	limiter := rate.NewLimiter(limiterRate, limiterBurst)

	for {
		// Wait until limiter allows request to happen
		if err := limiter.Wait(context.Background()); err != nil {
			log.Printf("[ERR] watch '%s': failed to wait for limiter", key)
			continue
		}

		// Make blocking query to watch key
		target := fmt.Sprintf("%s%s%s?index=%d", StringVal(s.cfg.ConsulAddr), StringVal(s.cfg.KVPath), key, index)
		resp, err := http.Get(target)
		if err != nil {
			log.Printf("[ERR] watch '%s': failed to get '%s': %v", key, target, err)
			continue
		}

		// Parse the raft index for this key (X-Consul-Index)
		header := resp.Header
		indexStr := header.Get("X-Consul-Index")
		if indexStr != "" {
			index, err = strconv.ParseUint(indexStr, 10, 64)
			if err != nil {
				log.Printf("[ERR] watch '%s': failed to parse X-Consul-Index: %v", key, err)
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
		lastIndex = index

		data := make([]keyResponse, 0)
		json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()

		// Key might not exist yet
		if len(data) == 0 {
			log.Printf("[WARN] watch '%s': empty response, key does not exist", key)
			continue
		}

		// We are not recursing on a key-prefix so these arrays will only return one value
		decoded, err := base64.StdEncoding.DecodeString(data[0].Value)
		if err != nil {
			log.Printf("[ERR] watch '%s': failed to decode value: '%s'", key, data[0].Value)
			continue
		}
		strVal := string(decoded)

		err = nil
		switch key {
		case "language":
			s.setLanguage(strVal)
		case StringVal(s.cfg.ServiceName) + "enable_checks":
			err = s.setEnableChecks(strVal)
		}
		if err != nil {
			log.Printf("[ERR] watch '%s': %v", key, err)
			continue
		}

		log.Printf("[INFO] watch '%s': updated to %s", key, strVal)
	}
}

func (s *server) setLanguage(lang string) {
	s.cfg.mu.Lock()
	defer s.cfg.mu.Unlock()

	s.cfg.Language = StringPtr(lang)
}

func (s *server) setEnableChecks(val string) error {
	s.cfg.mu.Lock()
	defer s.cfg.mu.Unlock()

	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fmt.Errorf("failed to parse enable_checks bool '%s': %v", val, err)
	}
	s.cfg.EnableChecks = BoolPtr(parsed)
	return nil
}

type keyResponse struct {
	LockIndex   uint64
	Key         string
	Flags       int
	Value       string
	CreateIndex uint64
	ModifyIndex uint64
}
