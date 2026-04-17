package ui

import (
        "context"
        "crypto/rand"
        "encoding/hex"
        "encoding/json"
        "fmt"
        "net"
        "net/http"
        "net/url"
        "os"
        "os/exec"
        "os/signal"
        "runtime"
        "strings"
        "sync"
        "sync/atomic"
        "syscall"
        "time"

        "github.com/TheAngelNerozzi/ghostoperator/internal/core"
        "github.com/TheAngelNerozzi/ghostoperator/internal/machine"
        "github.com/TheAngelNerozzi/ghostoperator/pkg/config"
)

// csrfToken is a random token generated once per server start for CSRF protection.
var csrfToken string

func init() {
        b := make([]byte, 32)
        if _, err := rand.Read(b); err != nil {
                // crypto/rand should never fail on modern OSes
                fmt.Fprintf(os.Stderr, "FATAL: cannot generate CSRF token: %v\n", err)
                os.Exit(1)
        }
        csrfToken = hex.EncodeToString(b)
}

// validateCSRF checks the double-submit cookie pattern for POST requests.
func validateCSRF(w http.ResponseWriter, r *http.Request) bool {
        if r.Method != http.MethodPost {
                return true // Only validate POST requests
        }
        // Check cookie
        cookie, err := r.Cookie("ghost_csrf")
        if err != nil {
                http.Error(w, "Missing CSRF token", http.StatusForbidden)
                return false
        }
        // Check header or form value
        headerToken := r.Header.Get("X-CSRF-Token")
        formToken := r.FormValue("csrf_token")
        submittedToken := headerToken
        if submittedToken == "" {
                submittedToken = formToken
        }
        if submittedToken == "" || submittedToken != cookie.Value || submittedToken != csrfToken {
                http.Error(w, "Invalid CSRF token", http.StatusForbidden)
                return false
        }
        return true
}

// csrfMiddleware wraps an http.HandlerFunc with CSRF validation for POST requests.
func csrfMiddleware(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method == http.MethodPost && !validateCSRF(w, r) {
                        return
                }
                next(w, r)
        }
}

// ShowDashboard launches the Ghost Mode web UI in the default browser.
func ShowDashboard(version string, cfg *config.AppConfig, m machine.Machine, onStart func(string, func(string)) error) {
        mux := http.NewServeMux()

        // Mutex to protect concurrent access to cfg
        var cfgMu sync.RWMutex

        var lastMetrics core.PulseMetrics

        // writeJSON is a helper that encodes JSON and sets the Content-Type header.
        writeJSON := func(w http.ResponseWriter, v interface{}) {
                w.Header().Set("Content-Type", "application/json")
                if err := json.NewEncoder(w).Encode(v); err != nil {
                        fmt.Printf("Warning: JSON encode error: %v\n", err)
                }
        }

        // isLoopbackURL checks whether a URL points to a loopback address (SSRF protection).
        // Uses net.IP.IsLoopback() to cover all loopback variants (127.0.0.0/8, ::1,
        // IPv4-mapped IPv6, etc.) and resolves hostnames via DNS to prevent rebinding.
        isLoopbackURL := func(rawURL string) bool {
                u, err := url.Parse(rawURL)
                if err != nil {
                        return false
                }
                scheme := strings.ToLower(u.Scheme)
                if scheme != "http" && scheme != "https" {
                        return false
                }
                host := u.Hostname()
                if host == "" {
                        return false
                }
                // Reject any URL containing a path, query, or fragment to prevent injection.
                if u.RawQuery != "" || u.Fragment != "" {
                        return false
                }
                ip := net.ParseIP(host)
                if ip != nil {
                        return ip.IsLoopback()
                }
                // For hostnames, resolve and check all addresses
                addrs, err := net.LookupIP(host)
                if err != nil {
                        return false
                }
                for _, addr := range addrs {
                        if !addr.IsLoopback() {
                                return false
                        }
                }
                return true
        }

        // Serve the main dashboard page
        mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                // Set CSRF cookie on page load
                http.SetCookie(w, &http.Cookie{
                        Name:     "ghost_csrf",
                        Value:    csrfToken,
                        Path:     "/",
                        HttpOnly: false, // Needs to be readable by JS for double-submit
                        SameSite: http.SameSiteStrictMode,
                })
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                fmt.Fprintf(w, dashboardHTML, version, version, version, csrfToken)
        })

        // missionActive tracks whether a mission is currently running to prevent concurrent execution.
        var missionActive int32

        // Mission execution endpoint (streams Server-Sent Events)
        mux.HandleFunc("/mission", csrfMiddleware(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "Method Not Allowed", 405)
                        return
                }

                // Limit request body to 1MB to prevent memory exhaustion
                r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

                mission := r.FormValue("intent")
                if mission == "" {
                        http.Error(w, "No intent provided", 400)
                        return
                }

                // Prevent concurrent mission execution
                if !atomic.CompareAndSwapInt32(&missionActive, 0, 1) {
                        http.Error(w, "Mission already in progress", http.StatusConflict)
                        return
                }
                defer atomic.StoreInt32(&missionActive, 0)

                ctx, cancel := context.WithCancel(r.Context())
                defer cancel()

                // Monitor client disconnect to stop mission
                go func() {
                        <-ctx.Done()
                        cancel()
                }()

                w.Header().Set("Content-Type", "text/event-stream")
                w.Header().Set("Cache-Control", "no-cache")
                w.Header().Set("Connection", "keep-alive")
                flusher, ok := w.(http.Flusher)
                if !ok {
                        http.Error(w, "Streaming not supported", 500)
                        return
                }

                err := onStart(mission, func(status string) {
                        defer func() {
                                if r := recover(); r != nil {
                                        fmt.Printf("Warning: panic in mission callback: %v\n", r)
                                }
                        }()
                        select {
                        case <-ctx.Done():
                                return
                        default:
                        }
                        fmt.Fprintf(w, "data: %s\n\n", status)
                        flusher.Flush()
                })

                if err != nil {
                        fmt.Fprintf(w, "data: ❌ Misión fallida: %v\n\n", err)
                } else {
                        fmt.Fprintf(w, "data: ✅ Misión completada.\n\n")
                }
                flusher.Flush()
        }))

        // Metrics endpoint for PhantomPulse
        mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
                cfgMu.RLock()
                defer cfgMu.RUnlock()
                writeJSON(w, lastMetrics)
        })

        // Health endpoint for Ollama status
        mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
                cfgMu.RLock()
                endpoint := cfg.OllamaEndpoint
                model := cfg.OllamaModel
                gridDensity := cfg.GridDensity
                cfgMu.RUnlock()

                // SSRF protection: only allow loopback addresses
                if !isLoopbackURL(endpoint) {
                        writeJSON(w, map[string]string{"status": "error", "model": model, "grid_density": gridDensity})
                        return
                }

                client := http.Client{Timeout: 1 * time.Second}
                resp, err := client.Get(endpoint + "/api/version")
                status := "ok"
                var ollamaVersion string
                if err != nil {
                        status = "error"
                } else {
                        defer resp.Body.Close()
                        var versionResp struct {
                                Version string `json:"version"`
                        }
                        if json.NewDecoder(resp.Body).Decode(&versionResp) == nil && versionResp.Version != "" {
                                ollamaVersion = versionResp.Version
                        }
                }
                writeJSON(w, map[string]string{"status": status, "model": model, "ollama_version": ollamaVersion, "grid_density": gridDensity})
        })

        // Hardware profile endpoint for fallback mode indicator
        mux.HandleFunc("/api/hardware", func(w http.ResponseWriter, r *http.Request) {
                cfgMu.RLock()
                defer cfgMu.RUnlock()
                profile := core.DetectHardwareProfile()
                writeJSON(w, map[string]interface{}{"is_weak": profile.IsWeak, "reason": profile.Reason, "total_ram": profile.TotalRAMBytes, "num_cpu": profile.NumCPU, "budget_ms": cfg.FallbackBudgetMs, "fallback_forced": cfg.HardwareFallback})
        })

        // Toggle fallback mode on/off and persist to config
        mux.HandleFunc("/api/fallback/toggle", csrfMiddleware(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "Method Not Allowed", 405)
                        return
                }
                cfgMu.Lock()
                cfg.HardwareFallback = !cfg.HardwareFallback
                cfgMu.Unlock()
                if err := cfg.Save(); err != nil {
                        http.Error(w, fmt.Sprintf("Failed to save config: %v", err), 500)
                        return
                }
                writeJSON(w, map[string]interface{}{"fallback_active": cfg.HardwareFallback, "budget_ms": cfg.FallbackBudgetMs})
        }))

        // Resume mission after interruption
        mux.HandleFunc("/api/resume", csrfMiddleware(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "Method Not Allowed", 405)
                        return
                }
                // Reset the machine's interruption state
                m.ResetIntervention()

                w.Header().Set("Content-Type", "application/json")
                json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
        }))

        // Find a free port
        listener, err := net.Listen("tcp", "127.0.0.1:7474")
        if err != nil {
                listener, err = net.Listen("tcp", "127.0.0.1:0")
                if err != nil {
                        fmt.Println("❌ Cannot start UI server:", err)
                        return
                }
        }

        addr := "http://" + listener.Addr().String()
        fmt.Printf("\033[1;32m[UI]\033[0m Ghost Mode UI → %s\n", addr)

        // Open browser after a short delay
        go func() {
                time.Sleep(500 * time.Millisecond)
                var cmd *exec.Cmd
                switch runtime.GOOS {
                case "windows":
                        cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", addr)
                case "darwin":
                        cmd = exec.Command("open", addr)
                default: // linux
                        cmd = exec.Command("xdg-open", addr)
                }
                if cmd != nil {
                        if err := cmd.Start(); err != nil {
                                fmt.Printf("Warning: could not open browser: %v\n", err)
                        }
                }
        }()

        // Graceful shutdown with signal handling
        srv := &http.Server{
                Handler:        mux,
                ReadTimeout:    10 * time.Second,
                WriteTimeout:   30 * time.Second,
                IdleTimeout:    60 * time.Second,
                MaxHeaderBytes: 1 << 20,
        }
        go func() {
                if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
                        fmt.Println("❌ UI server error:", err)
                }
        }()

        // Wait for interrupt signal
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit

        fmt.Println("\n\033[1;33m[UI]\033[0m Shutting down gracefully...")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := srv.Shutdown(ctx); err != nil {
                fmt.Println("❌ UI server forced shutdown:", err)
        }
        fmt.Println("\033[1;32m[UI]\033[0m Server stopped.")
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>GHOST OPERATOR v%s</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&family=JetBrains+Mono:wght@300;400&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#09090b;
  --surface:#111113;
  --surface2:#18181b;
  --border:#27272a;
  --border-light:#3f3f46;
  --text:#fafafa;
  --text-secondary:#a1a1aa;
  --text-dim:#52525b;
  --mono:'JetBrains Mono',monospace;
  --sans:'Inter',system-ui,-apple-system,sans-serif;
  --radius:10px;
  --radius-sm:6px;
  --transition:150ms cubic-bezier(.4,0,.2,1);
}
html,body{height:100%%;overflow:hidden}
body{
  background:var(--bg);
  color:var(--text);
  font-family:var(--sans);
  display:flex;
  flex-direction:column;
  -webkit-font-smoothing:antialiased;
  -moz-osx-font-smoothing:grayscale;
}

/* ── HEADER ── */
header{
  height:52px;
  padding:0 24px;
  border-bottom:1px solid var(--border);
  display:flex;
  align-items:center;
  justify-content:space-between;
  flex-shrink:0;
  background:var(--surface);
}
.header-left{display:flex;align-items:center;gap:12px}
.logo{
  font-size:11px;
  font-weight:600;
  letter-spacing:.18em;
  text-transform:uppercase;
  color:var(--text);
}
.logo .ver{
  font-size:9px;
  font-weight:400;
  color:var(--text-dim);
  margin-left:6px;
  letter-spacing:.08em;
}
.header-right{display:flex;align-items:center;gap:16px}
.health-badge{
  display:flex;
  align-items:center;
  gap:7px;
  padding:5px 12px;
  background:var(--surface2);
  border:1px solid var(--border);
  border-radius:20px;
  font-family:var(--mono);
  font-size:10px;
  letter-spacing:.06em;
  color:var(--text-secondary);
}
.health-dot{
  width:6px;height:6px;border-radius:50%%;
  background:var(--text-dim);
  transition:background var(--transition),box-shadow var(--transition);
}
.health-dot.ok{background:#a1a1aa;box-shadow:0 0 6px rgba(161,161,170,.3)}
.health-dot.err{background:#71717a;box-shadow:0 0 8px rgba(113,113,122,.4)}

/* ── LANG SELECT ── */
.lang-select{
  appearance:none;
  background:var(--surface2);
  border:1px solid var(--border);
  border-radius:var(--radius-sm);
  padding:4px 24px 4px 10px;
  color:var(--text-secondary);
  font-family:var(--sans);
  font-size:11px;
  font-weight:500;
  cursor:pointer;
  outline:none;
  transition:border-color var(--transition);
  background-image:url("data:image/svg+xml,%%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' fill='none'%%3E%%3Cpath d='M1 1l4 4 4-4' stroke='%%2371717a' stroke-width='1.4' stroke-linecap='round' stroke-linejoin='round'/%%3E%%3C/svg%%3E");
  background-repeat:no-repeat;
  background-position:right 8px center;
}
.lang-select:hover{border-color:var(--border-light)}
.lang-select:focus{border-color:var(--text-dim)}

/* ── LAYOUT ── */
main{
  flex:1;
  display:grid;
  grid-template-columns:260px 1fr;
  overflow:hidden;
}

/* ── SIDEBAR ── */
aside{
  border-right:1px solid var(--border);
  padding:24px 20px;
  display:flex;
  flex-direction:column;
  gap:28px;
  overflow-y:auto;
  background:var(--surface);
}
.sidebar-group{display:flex;flex-direction:column;gap:12px}
.sidebar-label{
  font-size:9px;
  font-weight:600;
  text-transform:uppercase;
  letter-spacing:.16em;
  color:var(--text-dim);
  padding:0 4px;
}
.info-card{
  background:var(--surface2);
  border:1px solid var(--border);
  border-radius:var(--radius);
  padding:14px 16px;
  display:flex;
  flex-direction:column;
  gap:10px;
}
.info-row{
  display:flex;
  justify-content:space-between;
  align-items:center;
  font-size:12px;
}
.info-row .label{color:var(--text-secondary);font-weight:400}
.info-row .value{color:var(--text);font-family:var(--mono);font-size:11px;font-weight:400}
.info-row .value.ok{color:var(--text-secondary)}
.info-row .value.err{color:var(--text-dim)}

/* ── CONSOLE ── */
.console{
  display:flex;
  flex-direction:column;
  position:relative;
  overflow:hidden;
}
#log{
  flex:1;
  overflow-y:auto;
  padding:28px 32px 110px 32px;
  font-family:var(--mono);
  font-size:12.5px;
  line-height:1.7;
  color:var(--text-secondary);
}
#log div{
  padding:3px 0 3px 14px;
  border-left:1px solid var(--border);
  margin-bottom:4px;
  transition:border-color var(--transition);
}
#log div:hover{border-left-color:var(--border-light)}
#log .sys{color:var(--text-dim);font-size:11px}
#log .cmd{color:var(--text)}
#log .res{color:var(--text-secondary)}
#log .err{color:#71717a}

/* ── INPUT BAR ── */
.input-bar{
  position:absolute;
  bottom:0;left:0;right:0;
  padding:20px 32px 24px 32px;
  background:linear-gradient(transparent,var(--bg) 40%%);
  display:flex;
  gap:10px;
  align-items:center;
}
.prompt{
  font-family:var(--mono);
  font-size:12px;
  color:var(--text-dim);
  user-select:none;
  flex-shrink:0;
}
#intent{
  flex:1;
  background:transparent;
  border:none;
  border-bottom:1px solid var(--border);
  padding:10px 0;
  color:var(--text);
  font-family:var(--mono);
  font-size:13px;
  outline:none;
  transition:border-color var(--transition);
}
#intent:focus{border-bottom-color:var(--text-secondary)}
#intent::placeholder{color:var(--text-dim)}
.exec-btn{
  background:var(--text);
  color:var(--bg);
  border:none;
  padding:9px 22px;
  border-radius:var(--radius-sm);
  font-family:var(--sans);
  font-size:11px;
  font-weight:600;
  letter-spacing:.08em;
  text-transform:uppercase;
  cursor:pointer;
  transition:opacity var(--transition);
  flex-shrink:0;
}
.exec-btn:hover{opacity:.85}
.exec-btn:active{opacity:.7}

/* ── MODAL ── */
#modal{
  position:fixed;inset:0;
  background:rgba(9,9,11,.92);
  backdrop-filter:blur(20px);
  display:none;
  flex-direction:column;
  align-items:center;
  justify-content:center;
  z-index:1000;
  text-align:center;
}
.modal-icon{
  width:56px;height:56px;
  border:1px solid var(--border-light);
  border-radius:50%%;
  display:flex;align-items:center;justify-content:center;
  margin-bottom:20px;
  color:var(--text-secondary);
  font-size:22px;
}
.modal-title{
  font-size:16px;
  font-weight:600;
  color:var(--text);
  letter-spacing:.02em;
  margin-bottom:8px;
}
.modal-desc{
  font-size:13px;
  color:var(--text-secondary);
  max-width:340px;
  line-height:1.5;
}
.modal-btn{
  margin-top:28px;
  background:var(--text);
  color:var(--bg);
  border:none;
  padding:11px 36px;
  border-radius:var(--radius-sm);
  font-family:var(--sans);
  font-size:12px;
  font-weight:600;
  letter-spacing:.06em;
  text-transform:uppercase;
  cursor:pointer;
  transition:opacity var(--transition);
}
.modal-btn:hover{opacity:.85}

/* ── SCROLLBAR ── */
::-webkit-scrollbar{width:3px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}
::-webkit-scrollbar-thumb:hover{background:var(--border-light)}

/* ── RESPONSIVE ── */
@media(max-width:800px){
  main{grid-template-columns:1fr}
  aside{display:none}
  header{padding:0 16px}
  #log{padding:20px 18px 100px 18px}
  .input-bar{padding:16px 18px 20px 18px}
}
</style>
</head>
<body>

<!-- Interruption Modal -->
<div id="modal">
  <div class="modal-icon">!</div>
  <div class="modal-title" data-i18n="modal_title">INTERRUPTION DETECTED</div>
  <div class="modal-desc" data-i18n="modal_desc">Mouse movement detected. Continue mission?</div>
  <button class="modal-btn" onclick="continuar()" data-i18n="modal_btn">Continue Mission</button>
</div>

<!-- Header -->
<header>
  <div class="header-left">
    <div class="logo">Ghost Operator<span class="ver">v%s</span></div>
  </div>
  <div class="header-right">
    <select class="lang-select" id="lang" onchange="setLang(this.value)">
      <option value="es">Espanol</option>
      <option value="en" selected>English</option>
      <option value="fr">Francais</option>
      <option value="zh">中文</option>
    </select>
    <div class="health-badge">
      <div class="health-dot" id="health-dot"></div>
      <span id="health-text" data-i18n="ai_checking">Checking AI...</span>
    </div>
  </div>
</header>

<!-- Main Layout -->
<main>
  <aside>
    <div class="sidebar-group">
      <div class="sidebar-label" data-i18n="sec_engine">Engine</div>
      <div class="info-card">
        <div class="info-row">
          <span class="label" id="ollama-label">Ollama</span>
          <span class="value" id="ollama-status">...</span>
        </div>
        <div class="info-row">
          <span class="label" id="model-label">Model</span>
          <span class="value" id="model-status">...</span>
        </div>
      </div>
    </div>
    <div class="sidebar-group">
      <div class="sidebar-label" data-i18n="sec_vision">Vision System</div>
      <div class="info-card">
        <div class="info-row">
          <span class="label" data-i18n="lbl_density">Density</span>
          <span class="value" id="grid-density">...</span>
        </div>
        <div class="info-row">
          <span class="label" data-i18n="lbl_grid">Grid Overlay</span>
          <span class="value ok" data-i18n="lbl_active">Active</span>
        </div>
      </div>
    </div>
    <div class="sidebar-group">
      <div class="sidebar-label" data-i18n="sec_system">System</div>
      <div class="info-card">
        <div class="info-row">
          <span class="label" data-i18n="lbl_platform">Platform</span>
          <span class="value" id="sys-platform">...</span>
        </div>
        <div class="info-row">
          <span class="label" data-i18n="lbl_hotkey">Hotkey</span>
          <span class="value">Alt+G</span>
        </div>
      </div>
    </div>
  </aside>

  <div class="console">
    <div id="log">
      <div class="sys">Ghost Operator v%s</div>
      <div class="sys" data-i18n="msg_ready">Ready. Awaiting instructions...</div>
    </div>
    <div class="input-bar">
      <span class="prompt">></span>
      <input type="hidden" id="csrf_token" value="%s">
      <input type="text" id="intent" data-i18n-placeholder="input_placeholder" placeholder="Enter a command..." autofocus>
      <button class="exec-btn" onclick="ejecutar()" data-i18n="btn_exec">Run</button>
    </div>
  </div>
</main>

<script>
/* ══════════════════════════════════════
   i18n — Internationalization System
   ══════════════════════════════════════ */
const i18n = {
  en: {
    ai_checking: "Checking AI...",
    ai_ready: "AI: Online",
    ai_offline: "AI: Offline",
    sec_engine: "Engine",
    sec_vision: "Vision System",
    sec_system: "System",
    lbl_density: "Density",
    lbl_grid: "Grid Overlay",
    lbl_active: "Active",
    lbl_platform: "Platform",
    lbl_hotkey: "Hotkey",
    msg_ready: "Ready. Awaiting instructions...",
    input_placeholder: "Enter a command...",
    btn_exec: "Run",
    modal_title: "INTERRUPTION DETECTED",
    modal_desc: "Mouse movement detected. Continue mission?",
    modal_btn: "Continue Mission",
    log_mission: "Mission:",
    log_resuming: "Resuming mission...",
    status_connected: "Connected",
    status_disconnected: "Disconnected",
    status_loaded: "Loaded",
    status_na: "N/A"
  },
  es: {
    ai_checking: "Verificando IA...",
    ai_ready: "IA: En linea",
    ai_offline: "IA: Desconectada",
    sec_engine: "Motor",
    sec_vision: "Sistema de Vision",
    sec_system: "Sistema",
    lbl_density: "Densidad",
    lbl_grid: "Cuadricula",
    lbl_active: "Activo",
    lbl_platform: "Plataforma",
    lbl_hotkey: "Atajo",
    msg_ready: "Listo. Esperando instrucciones...",
    input_placeholder: "Ingresa una orden...",
    btn_exec: "Ejecutar",
    modal_title: "INTERRUPCION DETECTADA",
    modal_desc: "Movimiento del raton detectado. Continuar mision?",
    modal_btn: "Continuar Mision",
    log_mission: "Mision:",
    log_resuming: "Reanudando mision...",
    status_connected: "Conectado",
    status_disconnected: "Desconectado",
    status_loaded: "Cargado",
    status_na: "N/A"
  },
  fr: {
    ai_checking: "Verification IA...",
    ai_ready: "IA: En ligne",
    ai_offline: "IA: Hors ligne",
    sec_engine: "Moteur",
    sec_vision: "Systeme de Vision",
    sec_system: "Systeme",
    lbl_density: "Densite",
    lbl_grid: "Grille",
    lbl_active: "Actif",
    lbl_platform: "Plateforme",
    lbl_hotkey: "Raccourci",
    msg_ready: "Pret. En attente d'instructions...",
    input_placeholder: "Entrez une commande...",
    btn_exec: "Executer",
    modal_title: "INTERRUPTION DETECTEE",
    modal_desc: "Mouvement de souris detecte. Continuer la mission?",
    modal_btn: "Continuer la Mission",
    log_mission: "Mission :",
    log_resuming: "Reprise de la mission...",
    status_connected: "Connecte",
    status_disconnected: "Deconnecte",
    status_loaded: "Charge",
    status_na: "N/A"
  },
  zh: {
    ai_checking: "正在检查 AI...",
    ai_ready: "AI: 在线",
    ai_offline: "AI: 离线",
    sec_engine: "引擎",
    sec_vision: "视觉系统",
    sec_system: "系统",
    lbl_density: "密度",
    lbl_grid: "网格覆盖",
    lbl_active: "已激活",
    lbl_platform: "平台",
    lbl_hotkey: "快捷键",
    msg_ready: "就绪。等待指令...",
    input_placeholder: "输入指令...",
    btn_exec: "执行",
    modal_title: "检测到中断",
    modal_desc: "检测到鼠标移动。是否继续任务?",
    modal_btn: "继续任务",
    log_mission: "任务:",
    log_resuming: "正在恢复任务...",
    status_connected: "已连接",
    status_disconnected: "已断开",
    status_loaded: "已加载",
    status_na: "N/A"
  }
};

let currentLang = localStorage.getItem('ghost_lang') || 'en';

function t(key) {
  return (i18n[currentLang] && i18n[currentLang][key]) || i18n.en[key] || key;
}

function setLang(lang) {
  currentLang = lang;
  localStorage.setItem('ghost_lang', lang);
  applyTranslations();
}

function applyTranslations() {
  document.querySelectorAll('[data-i18n]').forEach(el => {
    el.textContent = t(el.getAttribute('data-i18n'));
  });
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
    el.placeholder = t(el.getAttribute('data-i18n-placeholder'));
  });
  document.documentElement.lang = currentLang;
}

/* ══════════════════════════════════════
   Core Logic
   ══════════════════════════════════════ */
const log = document.getElementById('log');
const input = document.getElementById('intent');
const modal = document.getElementById('modal');
const csrfToken = document.getElementById('csrf_token').value;

// Detect platform
(function detectPlatform(){
  const ua = navigator.userAgent.toLowerCase();
  let p = 'Unknown';
  if(ua.includes('win')) p = 'Windows';
  else if(ua.includes('mac')) p = 'macOS';
  else if(ua.includes('linux')) p = 'Linux';
  document.getElementById('sys-platform').textContent = p;
})();

// Init language
document.getElementById('lang').value = currentLang;
applyTranslations();

input.addEventListener('keydown', e => { if(e.key === 'Enter') ejecutar(); });

function appendLog(text, cls) {
  const d = document.createElement('div');
  d.className = cls || '';
  d.textContent = text;
  log.appendChild(d);
  log.scrollTop = log.scrollHeight;
}

function continuar() {
  fetch('/api/resume', {method:'POST',headers:{'X-CSRF-Token':csrfToken}})
    .then(() => {
      modal.style.display = 'none';
      appendLog('>> ' + t('log_resuming'), 'cmd');
    });
}

function ejecutar() {
  const v = input.value.trim();
  if(!v) return;
  appendLog('>> ' + t('log_mission') + ' ' + v, 'cmd');
  input.value = '';

  const body = new URLSearchParams({intent: v, csrf_token: csrfToken});
  fetch('/mission', {method:'POST',body:body,headers:{'X-CSRF-Token':csrfToken}})
    .then(r => {
      const reader = r.body.getReader();
      const dec = new TextDecoder();
      function read() {
        reader.read().then(({done,value}) => {
          if(done) return;
          dec.decode(value).split('\n').forEach(l => {
            if(!l.startsWith('data: ')) return;
            const msg = l.slice(6);
            if(msg.includes('USER_INTERRUPTED')) {
              modal.style.display = 'flex';
            }
            appendLog(msg, msg.includes('Error') ? 'err' : 'res');
          });
          read();
        });
      }
      read();
    });
}

/* ══════════════════════════════════════
   Health Polling
   ══════════════════════════════════════ */
function updateHealth(d) {
  const dot = document.getElementById('health-dot');
  const text = document.getElementById('health-text');
  const ollamaLabel = document.getElementById('ollama-label');
  const ollamaStatus = document.getElementById('ollama-status');
  const modelLabel = document.getElementById('model-label');
  const modelStatus = document.getElementById('model-status');
  const gridDensity = document.getElementById('grid-density');

  if(d.status === 'ok') {
    dot.className = 'health-dot ok';
    text.textContent = t('ai_ready');
    ollamaStatus.textContent = t('status_connected');
    ollamaStatus.className = 'value ok';
    if(d.ollama_version) ollamaLabel.textContent = 'Ollama v' + d.ollama_version;
  } else {
    dot.className = 'health-dot err';
    text.textContent = t('ai_offline');
    ollamaStatus.textContent = t('status_disconnected');
    ollamaStatus.className = 'value err';
  }

  if(d.model) {
    modelLabel.textContent = d.model;
    modelStatus.textContent = d.status === 'ok' ? t('status_loaded') : t('status_na');
    modelStatus.className = 'value ' + (d.status === 'ok' ? 'ok' : 'err');
  }
  if(d.grid_density) gridDensity.textContent = d.grid_density;
}

setInterval(() => {
  fetch('/api/health').then(r=>r.json()).then(updateHealth).catch(()=>{});
}, 5000);
setTimeout(() => fetch('/api/health').then(r=>r.json()).then(updateHealth).catch(()=>{}), 800);
</script>
</body>
</html>`
