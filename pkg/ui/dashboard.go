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
        "sync"
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
func ShowDashboard(version string, cfg *config.AppConfig, m machine.Machine, onStart func(string, func(string))) {
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
        isLoopbackURL := func(rawURL string) bool {
                u, err := url.Parse(rawURL)
                if err != nil {
                        return false
                }
                h := u.Hostname()
                return h == "127.0.0.1" || h == "::1" || h == "localhost"
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
                fmt.Fprintf(w, dashboardHTML, version, version, csrfToken, version)
        })

        // Mission execution endpoint (streams Server-Sent Events)
        mux.HandleFunc("/mission", csrfMiddleware(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "Method Not Allowed", 405)
                        return
                }
                mission := r.FormValue("intent")
                if mission == "" {
                        http.Error(w, "No intent provided", 400)
                        return
                }

                w.Header().Set("Content-Type", "text/event-stream")
                w.Header().Set("Cache-Control", "no-cache")
                w.Header().Set("Connection", "keep-alive")
                flusher, ok := w.(http.Flusher)
                if !ok {
                        http.Error(w, "Streaming not supported", 500)
                        return
                }

                onStart(mission, func(status string) {
                        defer func() {
                                if r := recover(); r != nil {
                                        fmt.Printf("Warning: panic in mission callback: %v\n", r)
                                }
                        }()
                        fmt.Fprintf(w, "data: %s\n\n", status)
                        flusher.Flush()
                })

                fmt.Fprintf(w, "data: ✅ Misión completada.\n\n")
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
                        cmd.Start()
                }
        }()

        // Graceful shutdown with signal handling
        srv := &http.Server{Handler: mux}
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
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>GHOST OPERATOR v%s</title>
<link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
  :root {
    --bg: #050505;
    --surface: #0a0a0a;
    --border: #1a1a1a;
    --text: #ffffff;
    --muted: #888888;
    --accent: #555555;
    --success: #00ff88;
    --error: #ff4444;
    --radius: 12px;
  }
  
  * { box-sizing: border-box; margin: 0; padding: 0; }
  
  body { 
    background: var(--bg); 
    color: var(--text); 
    font-family: 'Outfit', sans-serif; 
    height: 100vh; 
    display: flex; 
    flex-direction: column; 
    overflow: hidden;
  }

  header {
    padding: 24px 40px;
    border-bottom: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .brand { font-weight: 700; font-size: 16px; letter-spacing: 0.2em; text-transform: uppercase; }
  .brand span { color: var(--muted); font-weight: 400; margin-left: 5px; font-size: 10px; }

  main {
    flex: 1;
    display: grid;
    grid-template-columns: 320px 1fr;
    overflow: hidden;
  }

  aside {
    border-right: 1px solid var(--border);
    padding: 40px;
    display: flex;
    flex-direction: column;
    gap: 40px;
  }
  .sidebar-section { display: flex; flex-direction: column; gap: 15px; }
  .section-title { font-size: 11px; text-transform: uppercase; letter-spacing: 0.15em; color: var(--muted); font-weight: 700; }
  
  .status-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .status-item { display: flex; justify-content: space-between; align-items: center; font-size: 13px; font-family: 'JetBrains Mono', monospace; }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: var(--success); }
  .dot.error { background: var(--error); box-shadow: 0 0 10px var(--error); }

  .terminal {
    background: rgba(0,0,0,0.5);
    display: flex;
    flex-direction: column;
    padding: 40px;
    position: relative;
    overflow: hidden;
  }
  #log {
    flex: 1;
    overflow-y: auto;
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    line-height: 1.6;
    color: #eee;
    padding-bottom: 100px;
  }
  #log div { margin-bottom: 12px; border-left: 2px solid var(--border); padding-left: 15px; }
  .mission-log { color: var(--success); }

  .input-container {
    position: absolute;
    bottom: 40px;
    left: 40px;
    right: 40px;
    display: flex;
    gap: 15px;
  }
  input {
    flex: 1;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 16px 24px;
    color: white;
    font-family: inherit;
    font-size: 15px;
    outline: none;
    transition: border-color 0.2s;
  }
  input:focus { border-color: var(--muted); }
  
  button {
    background: white;
    color: black;
    border: none;
    padding: 0 30px;
    border-radius: var(--radius);
    font-weight: 700;
    text-transform: uppercase;
    font-size: 12px;
    cursor: pointer;
    letter-spacing: 0.1em;
    transition: transform 0.1s;
  }
  button:active { transform: scale(0.98); }

  /* Interruption Modal */
  #modal {
    position: fixed; top: 0; left: 0; width: 100%%; height: 100%%;
    background: rgba(0,0,0,0.9); backdrop-filter: blur(10px);
    display: none; flex-direction: column; align-items: center; justify-content: center;
    z-index: 1000;
    text-align: center;
  }
  .modal-btn { background: #fff; color: #000; border: none; padding: 15px 40px; border-radius: 12px; font-weight: 700; margin-top: 30px; cursor: pointer; }

  ::-webkit-scrollbar { width: 4px; }
  ::-webkit-scrollbar-thumb { background: var(--border); }

  /* Responsive: collapse sidebar on small screens */
  @media (max-width: 768px) {
    main { grid-template-columns: 1fr; }
    aside { display: none; }
    header { padding: 16px 20px; }
    .terminal { padding: 20px; }
    .input-container { left: 20px; right: 20px; bottom: 20px; }
  }
</style>
</head>
<body>
<div id="modal">
  <div style="font-size: 48px; margin-bottom: 20px;">⚠️</div>
  <div style="font-weight: 700; font-size: 24px; color:white;">¡INTERRUPCIÓN DETECTADA!</div>
  <div style="color: var(--muted); margin-top: 10px;">Has movido el ratón. ¿Deseas que GhostOperator continúe?</div>
  <button class="modal-btn" onclick="continuar()">CONTINUAR MISIÓN</button>
</div>
<header>
  <div class="brand">GHOST OPERATOR <span>v%s</span></div>
  <div class="status-item" style="gap:10px;">
    <div id="health-dot" class="dot"></div>
    <span style="font-family:'JetBrains Mono'; font-size:11px; letter-spacing:0.1em;" id="health-text">LOCAL AI: LISTO</span>
  </div>
</header>

<main>
  <aside>
    <div class="sidebar-section">
      <div class="section-title">Hardware Profile</div>
      <div class="status-card">
        <div class="status-item">
          <span id="ollama-label">Ollama</span>
          <span id="ollama-status" style="color:var(--muted)">...</span>
        </div>
        <div class="status-item" id="model-container">
          <span id="model-label">Model</span>
          <span id="model-status" style="color:var(--muted)">...</span>
        </div>
      </div>
    </div>

    <div class="sidebar-section">
      <div class="section-title">Grid System</div>
      <div class="status-card">
        <div class="status-item">
          <span>Densidad</span>
          <span id="grid-density">...</span>
        </div>
        <div class="status-item">
          <span>Alpha-Numeric</span>
          <span style="color:var(--success)">Activo</span>
        </div>
      </div>
    </div>
  </aside>

  <div class="terminal">
    <div id="log">
      <div class="mission-log">GHOST CONSOLE v%s</div>
      <div>Esperando instrucciones de misión...</div>
    </div>
    <div class="input-container">
      <input type="hidden" id="csrf_token" value="%s">
      <input type="text" id="intent" placeholder="Orden natural (ej: 'abre chrome', 'busca gmail'...)" autofocus>
      <button onclick="ejecutar()">Ejecutar</button>
    </div>
  </div>
</main>

<script>
const log = document.getElementById('log');
const input = document.getElementById('intent');
const modal = document.getElementById('modal');
const csrfToken = document.getElementById('csrf_token').value;

input.addEventListener('keydown', e => { if(e.key === 'Enter') ejecutar(); });

function continuar() {
  fetch('/api/resume', {method: 'POST', headers: {'X-CSRF-Token': csrfToken}})
    .then(() => {
        modal.style.display = 'none';
        const nd = document.createElement('div');
        nd.className = 'mission-log';
        nd.textContent = '>> Reanudando misión...';
        log.appendChild(nd);
    });
}

function ejecutar() {
  const v = input.value.trim(); if(!v) return;
  const d = document.createElement('div');
  d.className = 'mission-log';
  d.textContent = '>> Misión: ' + v;
  log.appendChild(d);
  input.value = '';
  
  const body = new URLSearchParams({intent: v, csrf_token: csrfToken});
  fetch('/mission', {method: 'POST', body: body, headers: {'X-CSRF-Token': csrfToken}})
    .then(r => {
      const reader = r.body.getReader();
      const dec = new TextDecoder();
      function read() {
        reader.read().then(({done, value}) => {
          if(done) return;
          const content = dec.decode(value);
          const lines = content.split('\n');
          lines.forEach(l => {
            if(l.startsWith('data: ')) {
               const msg = l.slice(6);
               if (msg.includes('USER_INTERRUPTED')) {
                 modal.style.display = 'flex';
               }
               const nd = document.createElement('div');
               nd.textContent = msg;
               log.appendChild(nd);
               log.scrollTop = log.scrollHeight;
            }
          });
          read();
        });
      }
      read();
    });
}

// Check Local AI health regularly and update sidebar dynamically
setInterval(() => {
  fetch('/api/health')
    .then(r => r.json())
    .then(d => {
       const dot = document.getElementById('health-dot');
       const text = document.getElementById('health-text');
       const ollamaLabel = document.getElementById('ollama-label');
       const ollamaStatus = document.getElementById('ollama-status');
       const modelLabel = document.getElementById('model-label');
       const modelStatus = document.getElementById('model-status');
       const gridDensity = document.getElementById('grid-density');

       if(d.status === 'ok') {
         dot.className = 'dot';
         text.textContent = 'LOCAL AI: LISTO';
         ollamaStatus.textContent = 'Conectado';
         ollamaStatus.style.color = 'var(--success)';
         if(d.ollama_version) {
           ollamaLabel.textContent = 'Ollama v' + d.ollama_version;
         }
       } else {
         dot.className = 'dot error';
         text.textContent = 'OLLAMA: OFFLINE';
         ollamaStatus.textContent = 'Desconectado';
         ollamaStatus.style.color = 'var(--error)';
       }

       if(d.model) {
         modelLabel.textContent = d.model;
         modelStatus.textContent = d.status === 'ok' ? 'Cargado' : 'N/A';
       }
       if(d.grid_density) {
         gridDensity.textContent = d.grid_density;
       }
    });
}, 5000);

// Trigger initial health check
setTimeout(() => fetch('/api/health').then(r => r.json()).then(d => {
  const evt = new Event('poll');
  document.dispatchEvent(evt);
}), 1000);
</script>
</body>
</html>`
