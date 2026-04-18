package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"remote-text-input/internal/injector"
	"remote-text-input/internal/logger"
	"remote-text-input/internal/server"
	"remote-text-input/internal/util"
)

const (
	defaultHost       = "0.0.0.0"
	defaultPort       = 8765
	defaultDebugToken = "remote-text-input-debug"
)

type config struct {
	Host       string
	Port       int
	Token      string
	Debug      bool
	LogEnabled bool
}

const maxPortRetries = 10

func isPortAvailable(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func findAvailablePort(host string, startPort int) int {
	for i := 0; i < maxPortRetries; i++ {
		port := startPort + i
		if isPortAvailable(host, port) {
			return port
		}
	}
	return -1
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := injector.ValidateRuntime(); err != nil {
		log.Fatal(err)
	}

	logger.Verbose = cfg.LogEnabled

	availablePort := findAvailablePort(cfg.Host, cfg.Port)
	if availablePort == -1 {
		log.Fatalf("failed to find available port after %d attempts (starting from %d)", maxPortRetries, cfg.Port)
	}
	if availablePort != cfg.Port {
		log.Printf("port %d is in use, using port %d instead", cfg.Port, availablePort)
	}
	cfg.Port = availablePort

	displayHost := util.DetectLocalIP()
	localURL := fmt.Sprintf("http://127.0.0.1:%d/?token=%s", cfg.Port, cfg.Token)
	phoneURL := fmt.Sprintf("http://%s:%d/?token=%s", displayHost, cfg.Port, cfg.Token)
	pairURL := fmt.Sprintf("http://127.0.0.1:%d/pair?token=%s", cfg.Port, cfg.Token)

	inj := injector.New(cfg.LogEnabled, 80*time.Millisecond)

	app, err := server.NewApp(cfg.Token, cfg.LogEnabled, inj, localURL, phoneURL, pairURL)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	app.RegisterHandlers(mux)

	srv := &http.Server{Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), Handler: mux}

	fmt.Println("Remote Text Input MVP is running")
	fmt.Printf("Local URL : %s\n", localURL)
	fmt.Printf("Phone URL : %s\n", phoneURL)
	fmt.Printf("Pair URL  : %s\n", pairURL)
	fmt.Println("Send text from the phone page. The Linux side will type into terminals when possible, and fall back to paste elsewhere.")

	injector.PrintTerminalQR(phoneURL)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func readTokenFromConfig(path string) string {
	if path == "" {
		path = ".env"
	}
	data, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "TOKEN" || key == "RTI_TOKEN" {
			return strings.Trim(strings.TrimSpace(val), `"'`)
		}
	}
	return ""
}

func parseFlags() (config, error) {
	cfg := config{
		Host: defaultHost,
		Port: defaultPort,
	}
	var configPath string
	var randomToken bool

	flag.StringVar(&cfg.Host, "host", cfg.Host, "listen host")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "listen port")
	flag.StringVar(&cfg.Token, "token", "", "auth token")
	flag.StringVar(&configPath, "config", "", "path to .env config file")
	flag.BoolVar(&randomToken, "random", false, "generate random token")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug mode")
	flag.BoolVar(&cfg.LogEnabled, "log", false, "enable logging")
	flag.Parse()

	if cfg.Debug {
		cfg.LogEnabled = true
	}

	// Token resolution chain
	if cfg.Token == "" {
		cfg.Token = readTokenFromConfig(configPath)
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv("RTI_TOKEN")
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv("TOKEN")
	}
	if cfg.Token == "" && cfg.Debug {
		cfg.Token = defaultDebugToken
	}

	// Final check or random
	if cfg.Token == "" {
		if !randomToken {
			fmt.Println("Error: No authentication token provided.\nProvide a token via -token, .env file, or use -random for a random token.\n")
			flag.Usage()
			os.Exit(0)
		}
		var err error
		if cfg.Token, err = util.RandomToken(12); err != nil {
			return cfg, err
		}
	}

	return cfg, nil
}
