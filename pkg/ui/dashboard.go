package ui

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/TheAngelNerozzi/ghostoperator/internal/core"
	"github.com/TheAngelNerozzi/ghostoperator/pkg/config"
)

// ShowDashboard launches the Ghost Mode web UI in the default browser.
func ShowDashboard(version string, cfg *config.AppConfig, onStart func(string, func(string))) {
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
		client := &http.Client{Timeout: 30 * time.Second}
		_, err := client.Get("http://127.0.0.1:11434/api/version")
		status := "ok"
		if err != nil {
			status = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	})

	// Hardware profile endpoint for fallback mode indicator
	mux.HandleFunc("/api/hardware", func(w http.ResponseWriter, r *http.Request) {
		profile := core.DetectHardwareProfile()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_weak":         profile.IsWeak,
			"reason":          profile.Reason,
			"total_ram":       profile.TotalRAMBytes,
			"free_ram":        profile.FreeRAMBytes,
			"num_cpu":         profile.NumCPU,
			"budget_ms":       core.EffectiveBudgetMs(cfg.HardwareFallback, profile, cfg.FallbackBudgetMs),
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
		profile := core.DetectHardwareProfile()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fallback_active": cfg.HardwareFallback,
			"budget_ms":       core.EffectiveBudgetMs(cfg.HardwareFallback, profile, cfg.FallbackBudgetMs),
		})
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
		exec.Command("rundll32", "url.dll,FileProtocolHandler", addr).Start()
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
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
  :root {
    --bg: #0a0a0a;
    --surface: rgba(18, 18, 18, 0.92);
    --border: #222222;
    --text: #f0f0f0;
    --muted: #777777;
    --accent: #ffffff;
    --radius: 12px;
    --bubble-user: #1a1a1a;
    --bubble-ghost: #0d0d0d;
    --success: #22c55e;
    --error: #ef4444;
    --input-bg: rgba(20, 20, 20, 0.9);
    --placeholder: #555555;
    --btn-bg: #ffffff;
    --btn-text: #000000;
  }
  body.light-theme {
    --bg: #f8f9fb;
    --surface: rgba(255, 255, 255, 0.92);
    --border: #dde1e6;
    --text: #111827;
    --muted: #6b7280;
    --accent: #111827;
    --bubble-user: #e8eaee;
    --bubble-ghost: #ffffff;
    --input-bg: rgba(255, 255, 255, 0.95);
    --placeholder: #9ca3af;
    --btn-bg: #111827;
    --btn-text: #ffffff;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { 
    background: var(--bg); 
    color: var(--text); 
    font-family: 'Inter', sans-serif; 
    height: 100vh; 
    display: flex; 
    flex-direction: column; 
    align-items: center; 
    overflow: hidden;
    transition: background 0.3s, color 0.3s;
  }
  
  /* Header */
  header {
    width: 100%%;
    padding: 14px 24px;
    border-bottom: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: var(--surface);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    z-index: 10;
    position: sticky;
    top: 0;
    gap: 12px;
    flex-wrap: wrap;
  }
  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .brand { font-weight: 600; font-size: 15px; letter-spacing: -0.02em; display: flex; align-items: center; gap: 8px; white-space: nowrap; }
  .version-tag { font-family: 'JetBrains Mono', monospace; font-size: 10px; color: var(--muted); border: 1px solid var(--border); padding: 2px 6px; border-radius: 4px; }
  .header-divider { width: 1px; height: 18px; background: var(--border); flex-shrink: 0; }

  /* Chat Area */
  #log {
    flex: 1;
    width: 100%%;
    max-width: 800px;
    padding: 40px 24px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 24px;
    scroll-behavior: smooth;
  }
  .msg { max-width: 85%%; padding: 12px 16px; border-radius: var(--radius); font-size: 14px; line-height: 1.6; border: 1px solid transparent; animation: fadeIn 0.3s ease-out; }
  .msg-ghost { align-self: flex-start; background: var(--bubble-ghost); border-color: var(--border); }
  .msg-user { align-self: flex-end; background: var(--bubble-user); color: var(--text); }
  .msg-ghost .prefix { color: var(--muted); font-size: 10px; margin-bottom: 6px; font-weight: 500; text-transform: uppercase; letter-spacing: 0.08em; }

  @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }

  /* Input Bar */
  .input-container {
    width: 100%%;
    max-width: 800px;
    padding: 20px 24px 24px;
    background: linear-gradient(to top, var(--bg) 85%%, transparent);
  }
  .input-wrapper {
    background: var(--input-bg);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 6px 6px 6px 8px;
    display: flex;
    gap: 8px;
    align-items: center;
    transition: border-color 0.2s, box-shadow 0.2s;
    backdrop-filter: blur(8px);
  }
  .input-wrapper:focus-within { border-color: var(--muted); box-shadow: 0 0 0 3px rgba(255,255,255,0.04); }
  input#intent {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--text);
    font-family: inherit;
    font-size: 15px;
    padding: 10px 12px;
  }
  input#intent::placeholder { color: var(--placeholder); }

  /* Send Button (primary action) */
  .send-btn {
    background: var(--btn-bg);
    color: var(--btn-text);
    border: none;
    border-radius: 10px;
    padding: 10px 20px;
    font-weight: 600;
    font-size: 13px;
    cursor: pointer;
    transition: transform 0.1s, opacity 0.2s;
    letter-spacing: 0.02em;
    white-space: nowrap;
    font-family: inherit;
  }
  .send-btn:hover { opacity: 0.88; }
  .send-btn:active { transform: scale(0.96); }
  .send-btn:disabled { opacity: 0.2; cursor: not-allowed; }

  /* Controls (header-right) */
  .controls { display: flex; align-items: center; justify-content: center; gap: 10px; flex-shrink: 0; }
  .lang-select { 
    background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 6px; 
    padding: 5px 8px; font-size: 12px; cursor: pointer; font-family: 'JetBrains Mono', monospace; 
    font-weight: 500; outline: none; appearance: none; -webkit-appearance: none;
  }
  .lang-select option { background: var(--bg); color: var(--text); }
  .theme-toggle { 
    background: transparent; border: 1px solid var(--border); color: var(--text); font-size: 13px; 
    cursor: pointer; padding: 5px 10px; border-radius: 6px; display: flex; align-items: center; 
    justify-content: center; transition: background 0.2s; font-family: 'Inter', sans-serif; font-weight: 500;
    white-space: nowrap;
  }
  .theme-toggle:hover { background: rgba(128,128,128,0.1); }

  /* Footer & Status */
  .footer-status { font-size: 11px; color: var(--muted); letter-spacing: 0.05em; display: flex; align-items: center; gap: 6px; font-family: 'JetBrains Mono', monospace; white-space: nowrap; }
  .dot { width: 6px; height: 6px; border-radius: 50%%; background: var(--success); box-shadow: 0 0 6px var(--success); transition: background 0.3s, box-shadow 0.3s; flex-shrink: 0; }
  .dot.offline { background: var(--error); box-shadow: 0 0 6px var(--error); }

  /* Skeleton Loading */
  .skeleton-pulse {
    display: inline-block;
    width: 60px;
    height: 12px;
    border-radius: 4px;
    background: linear-gradient(90deg, var(--border) 25%%, var(--surface) 50%%, var(--border) 75%%);
    background-size: 200%% 100%%;
    animation: pulse 1.5s infinite;
  }
  @keyframes pulse { 0%% { background-position: 200%% 0; } 100%% { background-position: -200%% 0; } }

  /* Scrollbar */
  #log::-webkit-scrollbar { width: 4px; }
  #log::-webkit-scrollbar-thumb { background: var(--border); border-radius: 10px; }
  #log::-webkit-scrollbar-track { background: transparent; }

  /* Fallback Badge */
  .fallback-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 10px;
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.3);
    border-radius: 6px;
    padding: 3px 8px;
    background: rgba(245, 158, 11, 0.08);
    letter-spacing: 0.05em;
    animation: fadeIn 0.5s ease-out;
    white-space: nowrap;
  }
  body.light-theme .fallback-badge {
    color: #d97706;
    border-color: rgba(217, 119, 6, 0.4);
    background: rgba(245, 158, 11, 0.12);
  }
  .dot-amber {
    width: 6px;
    height: 6px;
    border-radius: 50%%;
    background: #f59e0b;
    box-shadow: 0 0 6px #f59e0b;
    animation: pulseAmber 2s infinite;
    flex-shrink: 0;
  }
  @keyframes pulseAmber {
    0%%, 100%% { opacity: 1; }
    50%% { opacity: 0.4; }
  }

  /* Fallback Toggle Button — fully isolated from base button */
  .fallback-toggle {
    background: transparent !important;
    border: 1px solid rgba(245, 158, 11, 0.3) !important;
    color: #f59e0b !important;
    font-size: 12px;
    cursor: pointer;
    padding: 3px 10px;
    border-radius: 6px;
    font-family: 'JetBrains Mono', monospace;
    transition: background 0.2s, border-color 0.2s;
    white-space: nowrap;
  }
  .fallback-toggle:hover { background: rgba(245, 158, 11, 0.12) !important; }
  .fallback-toggle.active { background: rgba(245, 158, 11, 0.18) !important; border-color: #f59e0b !important; }
  body.light-theme .fallback-toggle { color: #d97706 !important; border-color: rgba(217, 119, 6, 0.4) !important; }
  body.light-theme .fallback-toggle:hover { background: rgba(245, 158, 11, 0.15) !important; }
  body.light-theme .fallback-toggle.active { background: rgba(245, 158, 11, 0.22) !important; border-color: #d97706 !important; }

  /* Responsive */
  @media (max-width: 700px) {
    header { padding: 10px 16px; }
    .header-left { gap: 10px; }
    .header-divider { display: none; }
    #log { padding: 24px 16px; }
    .input-container { padding: 16px; }
  }
</style>
</head>
<body>
<header>
  <div class="header-left">
    <div class="brand">👻 GHOST OPERATOR <span class="version-tag">PRO v%s</span></div>
    <span class="header-divider"></span>
    <div class="footer-status" id="llm-status"><span class="dot" id="llm-dot"></span> <span id="llm-text">OLLAMA LISTO</span></div>
    <div class="fallback-badge" id="fallback-badge" style="display:none;">
      <span class="dot-amber"></span>
      <span id="fallback-text">FALLBACK</span>
    </div>
    <button class="fallback-toggle" id="fallback-toggle" style="display:none;" onclick="toggleFallback()" title="Toggle fallback mode">⚡</button>
  </div>
  <div class="controls">
    <select id="lang-select" class="lang-select" onchange="changeLang(this.value)">
      <option value="es">🇪🇸 ES</option>
      <option value="en">🇬🇧 EN</option>
    </select>
    <button id="theme-toggle" class="theme-toggle" onclick="toggleTheme()" aria-label="Cambiar tema">🌙 Dark</button>
  </div>
</header>

<div id="log">
  <div class="msg msg-ghost" id="ghost-intro">
    <div class="prefix">GHOST</div>
    ...
  </div>
</div>

<div class="input-container">
  <div class="input-wrapper">
    <input id="intent" type="text" placeholder="Escribe tu misión PhantomPulse™..." autofocus autocomplete="off">
    <button id="btn" class="send-btn" onclick="ejecutar()">ENVIAR</button>
  </div>
</div>

<script>
// UI Elements
const ghostIntro = document.getElementById('ghost-intro');
const themeToggle = document.getElementById('theme-toggle');
const llmDot = document.getElementById('llm-dot');
const llmText = document.getElementById('llm-text');
const log = document.getElementById('log');
const btn = document.getElementById('btn');
const input = document.getElementById('intent');

// Theme Management
let isLight = localStorage.getItem('ghost_theme') === 'light';
if (isLight) document.body.classList.add('light-theme');
updateThemeBtn();

function toggleTheme() {
  isLight = !isLight;
  document.body.classList.toggle('light-theme', isLight);
  localStorage.setItem('ghost_theme', isLight ? 'light' : 'dark');
  updateThemeBtn();
}

function updateThemeBtn() {
  if (typeof dict !== 'undefined' && currentLang) {
    themeToggle.innerHTML = isLight ? dict[currentLang].themeLight : dict[currentLang].themeDark;
  } else {
    themeToggle.innerHTML = isLight ? '☀️ Light' : '🌙 Dark';
  }
}

// Translations Dictionary (i18n)
let currentLang = localStorage.getItem('ghost_lang') || 'es';

const dict = {
  'es': {
    intro: 'PhantomPulse™ activo. ¿Qué deseas automatizar a la velocidad de la luz?',
    placeholder: 'Escribe tu misión PhantomPulse™...',
    send: 'ENVIAR',
    themeLight: '☀️ Claro',
    themeDark: '🌙 Oscuro',
    llmReady: 'OLLAMA LISTO',
    llmOffline: 'OLLAMA OFFLINE',
    uiOffline: 'UI OFFLINE',
    fbOn: '🐢 Fallback activado — budget ',
    fbOff: '⚡ Fallback desactivado — budget normal',
    fbTitleActive: 'Desactivar modo fallback',
    fbTitleInactive: 'Activar modo fallback'
  },
  'en': {
    intro: 'PhantomPulse™ active. What do you want to automate at the speed of light?',
    placeholder: 'Type your PhantomPulse™ mission...',
    send: 'SEND',
    themeLight: '☀️ Light',
    themeDark: '🌙 Dark',
    llmReady: 'OLLAMA READY',
    llmOffline: 'OLLAMA OFFLINE',
    uiOffline: 'UI OFFLINE',
    fbOn: '🐢 Fallback enabled — budget ',
    fbOff: '⚡ Fallback disabled — normal budget',
    fbTitleActive: 'Disable fallback mode',
    fbTitleInactive: 'Enable fallback mode'
  }
};

function changeLang(val) {
  currentLang = val;
  localStorage.setItem('ghost_lang', val);
  
  // Update static UI
  ghostIntro.innerHTML = '<div class="prefix">GHOST</div>' + dict[val].intro;
  input.placeholder = dict[val].placeholder;
  btn.textContent = dict[val].send;
  updateThemeBtn();
  document.getElementById('lang-select').value = val;
  
  // Update dynamic elements
  if (llmDot.classList.contains('offline')) {
    llmText.textContent = llmText.textContent.includes('UI') ? dict[val].uiOffline : dict[val].llmOffline;
  } else {
    llmText.textContent = dict[val].llmReady;
  }
}

// Initialize Language
changeLang(currentLang);

// Message handling
function addMsg(text, type) {
  const d = document.createElement('div');
  d.className = 'msg msg-' + type;
  if (type === 'ghost') d.innerHTML = '<div class="prefix">GHOST</div>' + text;
  else d.textContent = text;
  log.appendChild(d);
  log.scrollTop = log.scrollHeight;
}

input.addEventListener('keydown', e => { if(e.key === 'Enter') ejecutar(); });

function ejecutar() {
  const v = input.value.trim(); if(!v) return;
  addMsg(v, 'user');
  input.value = ''; btn.disabled = true;
  
  const body = new URLSearchParams({intent: v});
  
  fetch('/mission', {method: 'POST', body: body})
    .then(r => {
      const reader = r.body.getReader();
      const dec = new TextDecoder();
      let ghostMsg = document.createElement('div');
      ghostMsg.className = 'msg msg-ghost';
      ghostMsg.innerHTML = '<div class="prefix">GHOST</div>';
      log.appendChild(ghostMsg);
      
      let buffer = '';
      
      function read() {
        reader.read().then(({done, value}) => {
          if(done) { 
			btn.disabled = false; 
			let t = ghostMsg.querySelector('.skeleton-pulse'); if(t) t.remove(); 
			return; 
		  }
		  
          buffer += dec.decode(value, {stream: true});
		  let lines = buffer.split('\n');
		  buffer = lines.pop(); // keep incomplete line
		  
          lines.forEach(l => {
            if(l.startsWith('data: ')) {
               const content = l.slice(6);
               let sk = ghostMsg.querySelector('.skeleton-pulse');
               if (sk) sk.remove();

               if (content.includes('activo...')) {
                 ghostMsg.innerHTML += '<div>' + content + '<br><br><span class="skeleton-pulse"></span></div>';
               } else {
                 ghostMsg.innerHTML += '<div>' + content + '</div>';
               }
               log.scrollTop = log.scrollHeight;
            }
          });
          read();
        });
      }
      read();
    }).catch(() => {
      addMsg('Error de conexión local (PhantomPulse motor offline).', 'ghost');
      btn.disabled = false;
    });
}

// Ollama Health Check Loop
setInterval(() => {
	fetch('/api/health')
		.then(r => r.json())
		.then(data => {
			if(data.status === 'ok') {
				llmDot.classList.remove('offline');
				llmText.textContent = dict[currentLang].llmReady;
			} else {
				llmDot.classList.add('offline');
				llmText.textContent = dict[currentLang].llmOffline;
			}
		}).catch(()=> {
			llmDot.classList.add('offline');
			llmText.textContent = dict[currentLang].uiOffline;
		});
}, 5000);

// Hardware Fallback Detection (runs once on load)
function refreshFallback() {
  fetch('/api/hardware')
    .then(r => r.json())
    .then(data => {
      const badge = document.getElementById('fallback-badge');
      const text = document.getElementById('fallback-text');
      const toggle = document.getElementById('fallback-toggle');
      toggle.style.display = 'inline-flex';
      if (data.is_weak || data.fallback_forced) {
        badge.style.display = 'inline-flex';
        text.textContent = 'FALLBACK ' + data.budget_ms + 'ms';
        badge.title = data.reason || 'Manual';
        toggle.classList.add('active');
        toggle.textContent = '🐢 ON';
        toggle.title = dict[currentLang].fbTitleActive;
      } else {
        badge.style.display = 'none';
        toggle.classList.remove('active');
        toggle.textContent = '⚡ OFF';
        toggle.title = dict[currentLang].fbTitleInactive;
      }
    }).catch(() => {});
}
refreshFallback();

function toggleFallback() {
  fetch('/api/fallback/toggle', {method: 'POST'})
    .then(r => r.json())
    .then(data => {
      refreshFallback();
      addMsg(data.fallback_active
        ? dict[currentLang].fbOn + data.budget_ms + 'ms'
        : dict[currentLang].fbOff, 'ghost');
    }).catch(() => {});
}
</script>
</body>
</html>`
