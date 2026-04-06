package ui

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/TheAngelNerozzi/ghostoperator/internal/core"
	"github.com/TheAngelNerozzi/ghostoperator/internal/machine"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
)

// ShowDashboard launches the Ghost Mode web UI in the default browser.
func ShowDashboard(version string, cfg *config.AppConfig, m machine.Machine, onStart func(string, func(string))) {
	mux := http.NewServeMux()

	var lastMetrics core.PulseMetrics

	// Serve the main dashboard page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, dashboardHTML, version, version)
	})

	// Mission execution endpoint (streams Server-Sent Events)
	mux.HandleFunc("/mission", func(w http.ResponseWriter, r *http.Request) {
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
			defer func() { recover() }() // Prevent panic if writer is closed
			if w == nil {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", status)
			flusher.Flush()
		})

		fmt.Fprintf(w, "data: ✅ Misión completada.\n\n")
		flusher.Flush()
	})

	// Metrics endpoint for PhantomPulse
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lastMetrics)
	})

	// Health endpoint for Ollama status
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		client := http.Client{Timeout: 1 * time.Second}
		resp, err := client.Get(cfg.OllamaEndpoint + "/api/version")
		status := "ok"
		if err != nil {
			status = "error"
		} else {
			defer resp.Body.Close()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": status, "model": cfg.OllamaModel})
	})

	// Hardware profile endpoint for fallback mode indicator
	mux.HandleFunc("/api/hardware", func(w http.ResponseWriter, r *http.Request) {
		profile := core.DetectHardwareProfile()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_weak":         profile.IsWeak,
			"reason":          profile.Reason,
			"total_ram":       profile.TotalRAMBytes,
			"num_cpu":         profile.NumCPU,
			"budget_ms":       cfg.FallbackBudgetMs,
			"fallback_forced": cfg.HardwareFallback,
		})
	})

	// Toggle fallback mode on/off and persist to config
	mux.HandleFunc("/api/fallback/toggle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", 405)
			return
		}
		cfg.HardwareFallback = !cfg.HardwareFallback
		cfg.Save()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fallback_active": cfg.HardwareFallback,
			"budget_ms":       cfg.FallbackBudgetMs,
		})
	})

	// Resume mission after interruption
	mux.HandleFunc("/api/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", 405)
			return
		}
		// Reset the machine's interruption state
		m.ResetIntervention()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
	})

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

	if err := http.Serve(listener, mux); err != nil {
		fmt.Println("❌ UI server error:", err)
	}
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
    background: (0,0,0,0.5);
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
          <span>Ollama v0.1.30</span>
          <span style="color:var(--success)">Conectado</span>
        </div>
        <div class="status-item" id="model-container">
          <span>Moondream v2</span>
          <span style="color:var(--muted)">Cargado</span>
        </div>
      </div>
    </div>

    <div class="sidebar-section">
      <div class="section-title">Grid System</div>
      <div class="status-card">
        <div class="status-item">
          <span>Densidad</span>
          <span>20x20</span>
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
      <input type="text" id="intent" placeholder="Orden natural (ej: 'abre chrome', 'busca gmail'...)" autofocus>
      <button onclick="ejecutar()">Ejecutar</button>
    </div>
  </div>
</main>

<script>
const log = document.getElementById('log');
const input = document.getElementById('intent');
const modal = document.getElementById('modal');

input.addEventListener('keydown', e => { if(e.key === 'Enter') ejecutar(); });

function continuar() {
  fetch('/api/resume', {method: 'POST'})
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
  
  const body = new URLSearchParams({intent: v});
  fetch('/mission', {method: 'POST', body: body})
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

// Check Local AI health regularly
setInterval(() => {
  fetch('/api/health')
    .then(r => r.json())
    .then(d => {
       const dot = document.getElementById('health-dot');
       const text = document.getElementById('health-text');
       if(d.status === 'ok') {
         dot.className = 'dot';
         text.textContent = 'LOCAL AI: LISTO';
       } else {
         dot.className = 'dot error';
         text.textContent = 'OLLAMA: OFFLINE';
       }
    });
}, 5000);
</script>
</body>
</html>`
