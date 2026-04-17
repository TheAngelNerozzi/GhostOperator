package llm

import (
        "context"
        "fmt"
        "net/http"
        "net/url"
        "regexp"
        "strconv"
        "strings"

        "github.com/ollama/ollama/api"
)

// gridLabelRe is a pre-compiled regex for extracting grid labels from LLM responses.
// Package-level to avoid re-compilation on every call.
var gridLabelRe = regexp.MustCompile(`([A-Z]{1,2}[0-9]{1,2})`)

// noStream is a package-level variable used to disable streaming for Generate requests.
var noStream = false

// VisionClient handles communication with Ollama for UI reasoning.
type VisionClient struct {
        client   *api.Client
        model    string
        endpoint string
}

// NewVisionClient initializes a new Ollama client for vision tasks.
func NewVisionClient(endpoint, model string) (*VisionClient, error) {
        if model == "" {
                return nil, fmt.Errorf("model name must not be empty")
        }
        u, err := url.Parse(endpoint)
        if err != nil {
                return nil, fmt.Errorf("invalid endpoint URL: %w", err)
        }
        if u.Scheme != "http" && u.Scheme != "https" {
                return nil, fmt.Errorf("endpoint URL must use http or https scheme, got %q", u.Scheme)
        }
        if u.Host == "" {
                return nil, fmt.Errorf("endpoint URL must include a host")
        }
        c := api.NewClient(u, http.DefaultClient)
        return &VisionClient{client: c, model: model, endpoint: endpoint}, nil
}

// Ping checks if the Ollama server is reachable.
func (v *VisionClient) Ping(ctx context.Context) error {
        _, err := v.client.Version(ctx)
        if err != nil {
                return fmt.Errorf("Ollama is offline or unreachable at %s. Ensure Ollama is running", v.endpoint)
        }
        return nil
}

// sanitizeIntent removes potentially dangerous characters from user-provided intent
// to mitigate prompt injection attacks. It escapes double quotes and backslashes
// to prevent breaking out of quoted prompt context.
func sanitizeIntent(intent string) string {
        intent = strings.ReplaceAll(intent, "\\", "\\\\")
        intent = strings.ReplaceAll(intent, "\"", "\\\"")
        intent = strings.ReplaceAll(intent, "\n", " ")
        intent = strings.ReplaceAll(intent, "\r", " ")
        if len(intent) > 200 {
                intent = intent[:200]
        }
        return intent
}

// Reason takes a screenshot with grid, user intent, and returns the detected action.
func (v *VisionClient) Reason(ctx context.Context, imageData []byte, intent string) (string, error) {
        target := sanitizeIntent(intent)
        if target == "" {
                target = "the most salient interactive element on the screen"
        }

        prompt := fmt.Sprintf(`You are GhostOperator, a visual automation agent. 
Analyze the provided screenshot with an alphanumeric grid overlay (A1, B2, etc.).
TASK: Locate the following requested element: "%s"
RESPONSE: Return ONLY the grid cell label where the element is located (e.g. G5). Do not write anything else.`, target)

        req := &api.GenerateRequest{
                Model:  v.model,
                Prompt: prompt,
                Images: []api.ImageData{imageData},
                Stream: &noStream,
        }

        var sb strings.Builder
        err := v.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
                sb.WriteString(resp.Response)
                return nil
        })

        if err != nil {
                return "", fmt.Errorf("ollama reasoning failed: %w", err)
        }

        return v.extractLabel(sb.String())
}

// ReasonFast uses strict settings to force a short and fast response from the LLM
func (v *VisionClient) ReasonFast(ctx context.Context, imageData []byte, intent string) (string, error) {
        target := sanitizeIntent(intent)
        if target == "" {
                target = "element"
        }

        // Optimized prompt for minimum tokens layout processing
        prompt := fmt.Sprintf(`Find: "%s". Return ONLY the exact grid label (e.g. A1, B2). No explanation.`, target)

        req := &api.GenerateRequest{
                Model:  v.model,
                Prompt: prompt,
                Images: []api.ImageData{imageData},
                Stream: &noStream,
                Options: map[string]interface{}{
                        "num_predict": 10,  // Force very short response
                        "temperature": 0.1, // High confidence
                },
        }

        var sb strings.Builder
        err := v.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
                sb.WriteString(resp.Response)
                return nil
        })

        if err != nil {
                return "", fmt.Errorf("ollama fast-reasoning failed: %w", err)
        }

        return v.extractLabel(sb.String())
}

func (v *VisionClient) extractLabel(response string) (string, error) {
        // Sanitation: Extract label like A1, B22, etc.
        match := gridLabelRe.FindString(strings.ToUpper(response))
        if match == "" {
                return "", fmt.Errorf("could not find grid label in response: %s", response)
        }

        // Validate that the row number is >= 1 (grid labels are 1-based)
        colEnd := 0
        for colEnd < len(match) && match[colEnd] >= 'A' && match[colEnd] <= 'Z' {
                colEnd++
        }
        rowStr := match[colEnd:]
        row, err := strconv.Atoi(rowStr)
        if err != nil || row < 1 {
                return "", fmt.Errorf("invalid row in extracted label: %s", match)
        }

        return match, nil
}
