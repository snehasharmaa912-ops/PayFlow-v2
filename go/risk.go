package payflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const LargeChargeThreshold int64 = 100000

type riskEvalRequest struct {
	ID         string `json:"id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	CustomerID string `json:"customer_id"`
}

type riskEvalResponse struct {
	ChargeID string `json:"charge_id"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type RiskEvaluator interface {
	Evaluate(charge *Charge) (decision string, reason string, err error)
}

type ProcessRiskEvaluator struct {
	JavaPath string
	JarPath  string
	Timeout  time.Duration
}

func NewProcessRiskEvaluator(jarPath string) *ProcessRiskEvaluator {
	return &ProcessRiskEvaluator{
		JavaPath: "java",
		JarPath:  jarPath,
		Timeout:  5 * time.Second,
	}
}

func (p *ProcessRiskEvaluator) Evaluate(charge *Charge) (string, string, error) {
	req := riskEvalRequest{
		ID:         charge.ID,
		Amount:     charge.Amount,
		Currency:   charge.Currency,
		CustomerID: charge.CustomerID,
	}
	input, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("marshaling risk request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.JavaPath, "-jar", p.JarPath)
	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("running risk engine: %w (stderr: %s)", err, stderr.String())
	}

	var resp riskEvalResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &resp); err != nil {
		return "", "", fmt.Errorf("decoding risk engine response: %w (stdout: %s)", err, stdout.String())
	}

	return resp.Decision, resp.Reason, nil
}

func statusForDecision(decision string) string {
	switch decision {
	case "approve":
		return "succeeded"
	case "review":
		return "pending_review"
	case "decline":
		return "declined"
	default:
		return "pending_review"
	}
}
