// Package planner implements the Adaptive Planner for GhostOperator.
//
// The Adaptive Planner decides dynamically whether to use the EML
// mathematical engine (instant) or fall back to LLM inference (slow).
// It classifies incoming missions by complexity and routes them to the
// appropriate engine, providing maximum speed without sacrificing
// functionality for complex tasks.
package planner

import (
	"strings"
)

// EngineType indicates which computation engine to use.
type EngineType int

const (
	// EngineEML uses the mathematical EML framework (sub-microsecond).
	EngineEML EngineType = iota
	// EngineLLM falls back to Ollama LLM inference (1-2.5 seconds).
	EngineLLM
)

func (e EngineType) String() string {
	switch e {
	case EngineEML:
		return "EML"
	case EngineLLM:
		return "LLM"
	default:
		return "UNKNOWN"
	}
}

// Decision represents the planner's routing decision for a mission.
type Decision struct {
	Engine     EngineType
	Confidence float64 // 0.0-1.0 confidence in the EML route
	Reason     string  // Human-readable explanation
}

// MissionProfile holds extracted features from a mission intent.
type MissionProfile struct {
	Intent      string
	HasCoords   bool
	IsClick     bool
	IsType      bool
	IsNavigate  bool
	IsSimple    bool
	WordCount   int
	HasKeywords bool
}

// Planner is the adaptive routing engine that decides between EML and LLM.
type Planner struct {
	// Stats tracks routing statistics for dashboard display.
	Stats Stats
}

// Stats holds routing statistics.
type Stats struct {
	TotalMissions  int
	EMLRouted      int
	LLMRouted      int
	EMLSuccessRate float64
}

// New creates a new adaptive Planner.
func New() *Planner {
	return &Planner{}
}

// Classify analyzes a mission intent and returns a routing decision.
//
// Simple missions (click coordinates, type text, navigate to URL) are
// routed to EML for instant execution. Complex missions (find element,
// read content, semantic decisions) fall back to LLM.
func (p *Planner) Classify(intent string) Decision {
	p.Stats.TotalMissions++
	profile := extractProfile(intent)

	// Rule 1: Direct coordinate patterns → EML
	if profile.HasCoords {
		p.Stats.EMLRouted++
		return Decision{
			Engine:     EngineEML,
			Confidence: 0.95,
			Reason:     "Coordenadas directas detectadas en la mision",
		}
	}

	// Rule 2: Simple click/type with known targets → EML
	if profile.IsSimple && (profile.IsClick || profile.IsType) {
		p.Stats.EMLRouted++
		return Decision{
			Engine:     EngineEML,
			Confidence: 0.80,
			Reason:     "Mision simple de clic/escritura con objetivo claro",
		}
	}

	// Rule 3: Navigation to known URLs → EML
	if profile.IsNavigate {
		p.Stats.EMLRouted++
		return Decision{
			Engine:     EngineEML,
			Confidence: 0.85,
			Reason:     "Navegacion a URL detectada",
		}
	}

	// Rule 4: Short intents with common keywords → EML
	if profile.WordCount <= 5 && profile.HasKeywords {
		p.Stats.EMLRouted++
		return Decision{
			Engine:     EngineEML,
			Confidence: 0.70,
			Reason:     "Intento corto con palabras clave reconocidas",
		}
	}

	// Default: LLM fallback for complex/unknown missions
	p.Stats.LLMRouted++
	return Decision{
		Engine:     EngineLLM,
		Confidence: 0.0,
		Reason:     "Mision compleja o semantica, requiere推理 LLM",
	}
}

// extractProfile analyzes the mission intent text and extracts features.
func extractProfile(intent string) MissionProfile {
	profile := MissionProfile{
		Intent: intent,
	}

	lower := strings.ToLower(intent)
	words := strings.Fields(intent)
	profile.WordCount = len(words)

	// Check for coordinate patterns: (123, 456), [100x200], etc
	profile.HasCoords = strings.Contains(intent, "(") && strings.Contains(intent, ")") ||
		strings.Contains(intent, "[") && strings.Contains(intent, "]") ||
		containsDigitPair(lower)

	// Check for click-related keywords
	clickKeywords := []string{"click", "clic", "haz clic", "press", "pulsa",
		"selecciona", "select", "abre", "open", "cierra", "close",
		"minimiza", "maximiza", "boton", "button"}
	profile.IsClick = containsAny(lower, clickKeywords)

	// Check for type-related keywords
	typeKeywords := []string{"escribe", "type", "escribir", "escribo",
		"enter", "input", "texto", "text", "busca", "search",
		"buscar", "login", "password", "usuario"}
	profile.IsType = containsAny(lower, typeKeywords)

	// Check for navigation keywords
	navigateKeywords := []string{"ve a", "go to", "navega", "navigate",
		"url", "http", "www", "abrir pagina", "open page"}
	profile.IsNavigate = containsAny(lower, navigateKeywords)

	// Common keywords that indicate simple intent
	commonKeywords := []string{"guardar", "save", "cancelar", "cancel",
		"aceptar", "accept", "ok", "yes", "no", "siguiente", "next",
		"atras", "back", "forward", "adelante", "menu", "inicio",
		"home", "config", "settings", "archivo", "file", "editar",
		"edit", "copiar", "copy", "pegar", "paste", "cortar", "cut"}
	profile.HasKeywords = containsAny(lower, commonKeywords)

	// Simple: short intent with at least one keyword category
	profile.IsSimple = profile.WordCount <= 8 &&
		(profile.IsClick || profile.IsType || profile.HasKeywords)

	return profile
}

func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func containsDigitPair(s string) bool {
	// Check for patterns like "100,200" or "100 200" or "100x200"
	inDigit := false
	count := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			if !inDigit {
				inDigit = true
				count++
			}
		} else {
			inDigit = false
		}
	}
	return count >= 2
}
