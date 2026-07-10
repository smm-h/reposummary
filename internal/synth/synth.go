// Package synth turns a digest into prose via an explicitly-chosen LLM backend.
//
// There is NO silent fallback: if the chosen backend fails at runtime, that is
// a hard error. The caller picks "none", "claude-cli", or "anthropic-api"; the
// choice is honored or it errors.
package synth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/smm-h/reposummary/internal/digest"
	"github.com/smm-h/reposummary/internal/render"
)

// PROMPT_TEMPLATE instructs the model to write flowing prose from the digest.
const PROMPT_TEMPLATE = `You are writing a short journal entry narrating what happened in a software repository over a time window.

Write 2 to 4 short paragraphs of flowing prose. Describe the focus areas, notable landings, fixes, and the overall trajectory of the work. Be concrete and reference features by name where the digest names them.

Rules:
- Invent NOTHING that is not present in the digest below.
- Do not fabricate numbers, names, or outcomes.
- Output ONLY the prose. No preamble, no headings, no bullet lists, no closing remarks.

Here is the digest:`

// Synthesize produces narrative prose for a digest using the chosen backend.
func Synthesize(d digest.Digest, mode, model string) (string, error) {
	switch mode {
	case "none":
		return "", nil
	case "claude-cli":
		return synthesizeClaudeCLI(d, model)
	case "anthropic-api":
		return synthesizeAnthropicAPI(d, model)
	default:
		return "", fmt.Errorf("unknown synthesis mode: %s", mode)
	}
}

func buildPrompt(d digest.Digest) string {
	return PROMPT_TEMPLATE + "\n\n" + render.DigestForLLM(d)
}

func synthesizeClaudeCLI(d digest.Digest, model string) (string, error) {
	prompt := buildPrompt(d)
	cmd := exec.Command("claude", "-p", prompt, "--model", model)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	runErr := cmd.Run()

	stdout := strings.TrimSpace(out.String())
	combined := stdout + "\n" + errBuf.String()
	if runErr != nil || stdout == "" || strings.Contains(combined, "Not logged in") || strings.Contains(combined, "Invalid API key") {
		detail := strings.TrimSpace(errBuf.String())
		if detail == "" {
			if runErr != nil {
				detail = runErr.Error()
			} else {
				detail = "empty output"
			}
		}
		return "", fmt.Errorf("claude CLI synthesis failed: %s (run `claude` interactively to /login, or use --synthesis none)", detail)
	}
	return stdout, nil
}

// apiRequest is the Anthropic messages request body.
type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// apiResponse is the subset of the Anthropic messages response we read.
type apiResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func synthesizeAnthropicAPI(d digest.Digest, model string) (string, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return "", fmt.Errorf("anthropic-api synthesis requires ANTHROPIC_API_KEY to be set")
	}

	prompt := buildPrompt(d)
	reqBody := apiRequest{
		Model:     model,
		MaxTokens: 900,
		Messages:  []apiMessage{{Role: "user", Content: prompt}},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic-api: marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("anthropic-api: building request: %w", err)
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic-api: request failed: %w", err)
	}
	defer resp.Body.Close()

	var respBuf bytes.Buffer
	if _, err := respBuf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("anthropic-api: reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic-api: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(respBuf.String()))
	}

	var parsed apiResponse
	if err := json.Unmarshal(respBuf.Bytes(), &parsed); err != nil {
		return "", fmt.Errorf("anthropic-api: parsing response: %w", err)
	}
	if len(parsed.Content) == 0 {
		return "", fmt.Errorf("anthropic-api: response contained no content")
	}
	text := strings.TrimSpace(parsed.Content[0].Text)
	if text == "" {
		return "", fmt.Errorf("anthropic-api: response text was empty")
	}
	return text, nil
}
