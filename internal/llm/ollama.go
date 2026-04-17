package llm

import (
        "context"
        "fmt"
        "net/http"
        "net/url"
        "regexp"
        "strings"

        "github.com/ollama/ollama/api"
)

// VisionClient handles communication with Ollama for UI reasoning.
type VisionClient struct {
        client   *api.Client
        model    string
        endpoint string
}

// NewVisionClient initializes a new Ollama client for vision tasks.
func NewVisionClient(endpoint, model string) (*VisionClient, error) {
        u, err := url.Parse(endpoint)
        if err != nil {
                return nil, err
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

// Reason takes a screenshot with grid, user intent, and returns the detected action.
func (v *VisionClient) Reason(ctx context.Context, imageData []byte, intent string) (string, error) {
        // Pre-flight check
        if err := v.Ping(ctx); err != nil {
                return "", err
        }

        target := intent
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
                Stream: new(bool),
        }

        var response string
        err := v.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
                response += resp.Response
                return nil
        })

        if err != nil {
                return "", fmt.Errorf("ollama reasoning failed: %w", err)
        }

        return v.extractLabel(response)
}

// ReasonFast uses strict settings to force a short and fast response from the LLM
func (v *VisionClient) ReasonFast(ctx context.Context, imageData []byte, intent string) (string, error) {
        if err := v.Ping(ctx); err != nil {
                return "", err
        }

        target := intent
        if target == "" {
                target = "element"
        }

        // Optimized prompt for minimum tokens layout processing
        prompt := fmt.Sprintf(`Find: "%s". Return ONLY the exact grid label (e.g. A1, B2). No explanation.`, target)

        req := &api.GenerateRequest{
                Model:  v.model,
                Prompt: prompt,
                Images: []api.ImageData{imageData},
                Stream: new(bool),
                Options: map[string]interface{}{
                        "num_predict": 10,  // Force very short response
                        "temperature": 0.1, // High confidence
                },
        }

        var response string
        err := v.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
                response += resp.Response
                return nil
        })

        if err != nil {
                return "", fmt.Errorf("ollama fast-reasoning failed: %w", err)
        }

        return v.extractLabel(response)
}

func (v *VisionClient) extractLabel(response string) (string, error) {
        // Sanitation: Extract label like A1, B22, etc.
        re := regexp.MustCompile(`([A-Z]{1,2}[0-9]{1,2})`)
        match := re.FindString(strings.ToUpper(response))
        if match == "" {
                return "", fmt.Errorf("could not find grid label in response: %s", response)
        }
        return match, nil
}
