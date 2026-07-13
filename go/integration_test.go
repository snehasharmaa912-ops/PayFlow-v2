package payflow
import (
	"os"
	"testing"
)

func TestProcessRiskEvaluator_RealJar(t *testing.T) {
	jarPath := os.Getenv("PAYFLOW_RISK_JAR")
	if jarPath == "" {
		t.Skip("PAYFLOW_RISK_JAR not set; skipping real Scala integration test")
	}
	if _, err := os.Stat(jarPath); err != nil {
		t.Skipf("risk engine jar not found at %s: %v", jarPath, err)
	}

	evaluator := NewProcessRiskEvaluator(jarPath)
	charge := &Charge{ID: "ch_test", Amount: 2000000, Currency: "USD", CustomerID: "cus_1"}

	decision, reason, err := evaluator.Evaluate(charge)
	if err != nil {
		t.Fatalf("evaluating charge against real jar: %v", err)
	}
	if decision != "decline" {
		t.Errorf("expected decline for a $20,000 charge, got %q (%s)", decision, reason)
	}
}
