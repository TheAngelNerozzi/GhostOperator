<p align="center">
  <img src="logo.svg" width="200" alt="GhostOperator Logo" />
</p>

<h1 align="center">GhostOperator (GO)</h1>

<p align="center">
  <strong>The local-first action agent that sees what you see, without APIs.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/github/actions/workflow/status/TheAngelNerozzi/ghostoperator/release.yml?style=for-the-badge" alt="Build Status" />
  <img src="https://img.shields.io/github/license/TheAngelNerozzi/ghostoperator?style=for-the-badge" alt="License" />
  <img src="https://img.shields.io/github/downloads/TheAngelNerozzi/ghostoperator/total?style=for-the-badge" alt="Downloads" />
  <a href="https://discord.gg/ghostoperator"><img src="https://img.shields.io/discord/1234567890?style=for-the-badge&label=Discord&logo=discord&logoColor=white" alt="Discord" /></a>
</p>

---

## ⚡️ Zero-Friction Installation

Get up and running in seconds. No Python, no C++, no cloud keys.

### Windows (PowerShell)
```powershell
irm https://get.ghostoperator.ai | iex
```

### macOS / Linux (Bash)
```bash
curl -sSL https://get.ghostoperator.ai | sh
```

---

## 🧠 How it Works

GhostOperator acts as a high-speed neural bridge between multimodal AI models and your operating system.

```mermaid
graph LR
    Screen[Monitor] -->|Capture| P1(pkg/screen)
    P1 -->|Grid Overlay| LMM{Local Brain}
    LMM -->|Action JSON| P2(pkg/action)
    P2 -->|Native Syscall| OS[OS Interface]
    
    style LMM fill:#00F0FF,stroke:#333,stroke-width:2px,color:#000
    style OS fill:#1A1A1A,stroke:#00F0FF,stroke-width:2px,color:#fff
```

---

## 🚀 Key Features

| Feature | Description |
| :--- | :--- |
| **🛡️ Privacy-First** | Zero cloud, zero telemetry. Your data never leaves your RAM. Optimized for local LMMs like Ollama. |
| **🏁 Grid Vision** | Advanced alphanumeric grid (A1, B2...) allows even the smallest AI models (Phi-3, Moondream) to hit targets with 100% precision. |
| **💨 Native Speed** | Built in pure Go. Sub-100ms latency from screen capture to action execution. No overhead, no interpreters. |
| **🛑 Safety Built-in** | Hardware-level Kill-Switch. Move your mouse or hit `Esc` to instantly regain manual control. |

---

## 🛠 Features for Developers

GhostOperator is designed to be highly extensible. You can build "Skills" that automate complex workflows (e.g., "Check my email and summarize Jira").

- **Modular Architecture**: Core logic in `/pkg`, easily importable.
- **Action Protocol**: Standardized JSON-RPC schema for easy integration with any LLM.
- **CGO-Free**: Compile to a single static binary on any platform.

---

## 🤝 Contributing

We are building the future of open-source automation. Whether it's adding a new OS syscall, optimizing the Grid system, or creating a new Skill, your contribution is welcome!

1. Star the repo.
2. Fork GhostOperator.
3. Check out [CONTRIBUTING.md](CONTRIBUTING.md).

---

<p align="center">
  Built with ❤️ by the GhostOperator Team.<br/>
  <i>Empowering humans with invisible automation.</i>
</p>
