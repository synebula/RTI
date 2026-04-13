package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"remote-text-input/internal/logger"
	"remote-text-input/internal/util"
)

type Injector interface {
	CommitText(text string, pressEnter bool, preferTerminalPaste bool) (string, error)
	SendEnter() error
}

type ServerApp struct {
	Token    string
	Injector Injector
	Log      bool
	localURL string
	phoneURL string
	pairURL  string
	rootHTML string
	pairHTML string
	qrJS     string
}

type commitRequest struct {
	Text      string `json:"text"`
	Enter     bool   `json:"enter"`
	InputMode string `json:"inputMode"`
}

type jsonResponse map[string]any

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func writeJSON(w http.ResponseWriter, status int, data jsonResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func NewApp(token string, logEnabled bool, inj Injector, localURL, phoneURL, pairURL string) (*ServerApp, error) {
	pages, err := LoadPages()
	if err != nil {
		return nil, err
	}
	pairHTML, err := pages.RenderPairPage(localURL, phoneURL)
	if err != nil {
		return nil, err
	}
	return &ServerApp{
		Token:    token,
		Injector: inj,
		Log:      logEnabled,
		localURL: localURL,
		phoneURL: phoneURL,
		pairURL:  pairURL,
		rootHTML: pages.rootHTML,
		pairHTML: pairHTML,
		qrJS:     pages.qrJS,
	}, nil
}

func (a *ServerApp) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/", a.handleRoot)
	mux.HandleFunc("/pair", a.handlePair)
	mux.HandleFunc("/qrcode.min.js", a.handleQRJS)
	mux.HandleFunc("/api/commit", a.handleCommit)
	mux.HandleFunc("/api/enter", a.handleEnter)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
}

func (a *ServerApp) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeJSON(w, http.StatusNotFound, jsonResponse{"error": "not found"})
		return
	}
	if !a.authorizePage(w, r) {
		return
	}
	logger.Event("page-open", "client", util.QuotedClientIP(r), "path", r.URL.Path)
	writeHTML(w, http.StatusOK, a.rootHTML)
}

func (a *ServerApp) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/pair" {
		writeJSON(w, http.StatusNotFound, jsonResponse{"error": "not found"})
		return
	}
	if !a.authorizePage(w, r) {
		return
	}
	logger.Event("page-open", "client", util.QuotedClientIP(r), "path", r.URL.Path)
	writeHTML(w, http.StatusOK, a.pairHTML)
}

func (a *ServerApp) handleQRJS(w http.ResponseWriter, r *http.Request) {
	if a.qrJS == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, a.qrJS)
}

func (a *ServerApp) handleCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{"error": "method not allowed"})
		return
	}
	if !a.authorizeAPI(w, r) {
		return
	}
	var req commitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, jsonResponse{"error": "invalid json"})
		return
	}
	fields := []string{
		"client", util.QuotedClientIP(r),
		"chars", fmt.Sprintf("%d", len(req.Text)),
		"enter", fmt.Sprintf("%t", req.Enter),
		"mode", fmt.Sprintf("%q", req.InputMode),
	}
	if a.Log {
		fields = append(fields, "text", fmt.Sprintf("%q", util.FormatTextPreview(req.Text, 120)))
	}
	logger.Event("commit-request", fields...)
	usedMode, err := a.Injector.CommitText(req.Text, req.Enter, req.InputMode == "terminal")
	if err != nil {
		logger.Event("inject-error", "client", util.QuotedClientIP(r), "path", r.URL.Path, "error", fmt.Sprintf("%q", err.Error()))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{"error": err.Error()})
		return
	}
	logger.Event("commit-ok", "client", util.QuotedClientIP(r), "chars", fmt.Sprintf("%d", len(req.Text)), "enter", fmt.Sprintf("%t", req.Enter), "mode", fmt.Sprintf("%q", usedMode))
	writeJSON(w, http.StatusOK, jsonResponse{"ok": true, "mode": usedMode})
}

func (a *ServerApp) handleEnter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, jsonResponse{"error": "method not allowed"})
		return
	}
	if !a.authorizeAPI(w, r) {
		return
	}
	logger.Event("enter-request", "client", util.QuotedClientIP(r))
	if err := a.Injector.SendEnter(); err != nil {
		logger.Event("inject-error", "client", util.QuotedClientIP(r), "path", r.URL.Path, "error", fmt.Sprintf("%q", err.Error()))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{"error": err.Error()})
		return
	}
	logger.Event("enter-ok", "client", util.QuotedClientIP(r))
	writeJSON(w, http.StatusOK, jsonResponse{"ok": true})
}

func (a *ServerApp) authorizePage(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("token") != a.Token {
		logger.Event("page-denied", "client", util.QuotedClientIP(r), "path", r.URL.Path)
		writeHTML(w, http.StatusUnauthorized, "<h1>401</h1><p>token 不正确。</p>")
		return false
	}
	return true
}

func (a *ServerApp) authorizeAPI(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Auth-Token") != a.Token {
		logger.Event("api-denied", "client", util.QuotedClientIP(r), "path", r.URL.Path)
		writeJSON(w, http.StatusUnauthorized, jsonResponse{"error": "unauthorized"})
		return false
	}
	return true
}
