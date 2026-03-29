<div align="center">

# 👻 GhostOperator (GO) v1.0
**The High-Performance Visual Automation Agent for Desktop Ecosystems**  
[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/TheAngelNerozzi/ghostoperator)](https://goreportcard.com/report/github.com/TheAngelNerozzi/ghostoperator)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

[**Website**](https://github.com/TheAngelNerozzi/ghostoperator) • [**Documentation**](#-features) • [**Install**](#-installation--downloads)

</div>

---

GhostOperator is a highly optimized, fully open-source autonomous agent designed for seamless integration with local AI models (LLMs). It interacts with your computer physically (Mouse/Keyboard) identically to how a human does. By natively seeing your screen without APIs or network latency, GhostOperator automates workflows securely on your local machine using state-of-the-art vision models.

## 🚀 Features

- 🧠 **Ollama Integration**: Reasoning-powered automation parsing using local models like Moondream.
- ⚡ **PhantomPulse™ Engine**: Ultra-fast image compression and grid resolution adaptation that adjusts dynamically to your hardware.
- 🐢 **Native Hardware Fallback**: A built-in adaptive mechanism that accurately detects slow or weak core CPUs, increasing operation budgets and decoupling execution limits automatically to prevent crashes.
- 🎯 **Grid Vision System™**: Alphanumeric coordinate mapping for sub-pixel precision and native smart actions (`DOUBLE_CLICK`, `CLICK`, `TYPE`).
- 🌗 **Ghost Mode UI**: A gorgeous minimalist monochrome HTTP dashboard featuring light/dark mode toggles and real-time hardware telemetry feedback (127.0.0.1:7474).
- 🖱️ **Organic "Ghost Glide" Simulation**: Moves the mouse with human-like cubic deceleration, achieving realistic UI interactions that gracefully bypass robotic detectors.

## 🏗 Architecture Focus

Built explicitly without heavy CGO linkages to keep it 100% portable:
- `/cmd/ghost`: Core executable and CLI orchestration bindings.
- `/internal/vision`: Vision engine capable of compressing arrays seamlessly and Grid System rendering.
- `/internal/automation`: Precise mechanical input simulation via OS system calls.
- `/internal/llm`: The localized Reasoning client routing to Ollama APIs.
- `/pkg/ui`: HTML/JS Dashboard with local state persistence configurations (`config.json`).

---

## 💻 System Requirements

GhostOperator’s performance is directly bound to your hardware's capability to run Local LLMs (Ollama + Moondream). 

| Component | 🐢 Minimum (Fallback Mode Active) | ⚡ Recommended (Real-Time Fluid) |
| :--- | :--- | :--- |
| **OS** | Windows 10/11 (64-bit) | Windows 11 / AWS EC2 (`g4dn.xlarge`) |
| **CPU** | 2 Cores (e.g., Celeron, old i3) | 8+ Cores (e.g., Core i7, Ryzen 7) |
| **GPU** | Integrated Intel/AMD Graphics | **Dedicated Nvidia** (RTX 3060+, T4, A100) |
| **RAM** | 8 GB | 16 GB+ |
| **Pacing** | ~2 to 20 mins per action *(Be patient!)* | **~1.5 to 2.5 seconds** per action |

---

## 📥 Installation & Downloads

GhostOperator relies exclusively on [Ollama](https://ollama.com/) to process vision locally. For your privacy, not a single snapshot leaves your network. 

### <img src="https://upload.wikimedia.org/wikipedia/commons/e/e0/Git-logo.svg" height="20" align="absmiddle" /> Quick Setup Wizard
All platforms can perform a quick installation wizard:

```bash
git clone https://github.com/TheAngelNerozzi/ghostoperator
cd ghostoperator
```

### <img src="https://upload.wikimedia.org/wikipedia/commons/4/48/Markdown-mark.svg" height="20" align="absmiddle" /> Platform Binaries
*GhostOperator automatically asks to install **Moondream 1.8B** (~940MB, ultra-lightweight and efficient for modern CPUs/integrative GPUs) during initial setup!*

<details open>
<summary><b>🟦 Windows (Native)</b></summary>

Currently, the primary supported operating system with native `user32.dll` acceleration logic.

```powershell
# Run the automated PowerShell installer
.\setup.ps1

# Or build natively from source
go build -ldflags "-s -w" -o ghost.exe ./cmd/ghost
.\ghost.exe
```
</details>

<details>
<summary><b>🍎 macOS (Apple Silicon / Intel)</b></summary>

*Note: Automation bindings for MacOS are under active community development. Setup primarily boots the core web UI and engine tests.*

```bash
# Run the bash installer
chmod +x setup.sh
./setup.sh

# Or build from source
go build -o ghost ./cmd/ghost
./ghost
```
</details>

<details>
<summary><b>🐧 Linux (X11 / Wayland)</b></summary>

*Note: Requires `xdotool` or specific Wayland compositor permissions to cast ghost movements.*

```bash
# General Linux Installation
chmod +x setup.sh
./setup.sh

# Compiling binaries
go build -o ghost ./cmd/ghost
./ghost
```
</details>

---

## 📖 User Manual & Prompting Guide

Once the Ghost Dashboard is running on `127.0.0.1:7474`, you simply talk to it. The **Action Modifier Engine** reads your intent to interact accurately with the OS.

### 1. Opening Desktop Apps (Double-Clicks)
Currently, a single mouse click on a Windows desktop icon only *selects* it. If you want into launch applications or open files, use the **Abre** (open) verb to instantly trigger the `DOUBLE_CLICK` protocol:
- ✅ *"Abre la papelera de reciclaje"*
- ✅ *"Abre el navegador Firefox"*
- ✅ *"Abre la carpeta de descargas"*

### 2. General Web & UI Interaction (Single Clicks)
For generic OS navigation, browser usage, or dismissing popups, simply phrase it naturally:
- ✅ *"Haz click en el botón de aceptar"*
- ✅ *"Busca el navegador y dale click"*
- ✅ *"Cierra la ventana"*

---

## 🧪 Testing and CI

We value clean code and robust mechanics. GhostOperator is rigorously tested over 80% coverage on internal modules:

```bash
# Run the full test suite
go test ./... -v -count=1

# Run benchmarks
go test ./internal/core/... -bench=.
```

## 🤝 Roadmap (v2.0 Beta Planning)
- [ ] Integration with multi-step reasoning models via LangChain/Go.
- [ ] Adding native `libxdo` for Linux and `CGEvent` for MacOS core.
- [ ] Full memory tracking for cross-app session retention.

<br>

<div align="center">
  Built by <b>Angel Nerozzi - Open Source</b> 🛸✨👻<br>
  <i>"No machine can replace the human spark, but it shouldn't have to push the buttons either."</i>
</div>
