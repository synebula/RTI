package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
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
	Host             string
	Port             int
	Token            string
	Debug            bool
	DryRun           bool
	LogEnabled       bool
	RestoreClipboard bool
	PasteDelay       time.Duration
	OpenPairPage     bool
	PrintTerminalQR  bool
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

	inj := injector.New(cfg.DryRun, cfg.LogEnabled, cfg.RestoreClipboard, cfg.PasteDelay)

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
	fmt.Printf("Mode      : %s\n", map[bool]string{true: "dry-run", false: "live"}[cfg.DryRun])
	fmt.Println("Send text from the phone page. The Linux side will type into terminals when possible, and fall back to paste elsewhere.")
	if cfg.PrintTerminalQR {
		injector.PrintTerminalQR(phoneURL)
	}
	if cfg.OpenPairPage {
		injector.OpenBrowserPage(pairURL)
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func parseFlags() (config, error) {
	var cfg config
	var token string
	var pasteDelayMS int
	var noRestore bool
	var noTerminalQR bool

	flag.StringVar(&cfg.Host, "host", defaultHost, "listen host")
	flag.IntVar(&cfg.Port, "port", defaultPort, "listen port")
	flag.StringVar(&token, "token", "", "auth token")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug mode with stable token and runtime logs")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "log injection commands instead of running them")
	flag.BoolVar(&cfg.LogEnabled, "log", false, "display runtime logs including text payloads")
	flag.BoolVar(&noRestore, "no-restore-clipboard", false, "do not restore clipboard after paste")
	flag.IntVar(&pasteDelayMS, "paste-delay-ms", 80, "delay between paste and enter in milliseconds")
	flag.BoolVar(&cfg.OpenPairPage, "pair", false, "open the pairing page in the default browser")
	flag.BoolVar(&noTerminalQR, "no-terminal-qr", false, "do not print the phone QR code in the terminal")
	flag.Parse()

	if cfg.Debug {
		cfg.LogEnabled = true
	}

	if token == "" && cfg.Debug {
		token = defaultDebugToken
	}

	if token == "" {
		generated, err := util.RandomToken(12)
		if err != nil {
			return cfg, err
		}
		token = generated
	}
	cfg.Token = token
	cfg.RestoreClipboard = !noRestore
	cfg.PrintTerminalQR = !noTerminalQR
	cfg.PasteDelay = time.Duration(pasteDelayMS) * time.Millisecond
	return cfg, nil
}
