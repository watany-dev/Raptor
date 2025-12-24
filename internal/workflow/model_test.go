package workflow

import "testing"

func TestStep_IsAction(t *testing.T) {
	t.Run("returns true when Uses is set", func(t *testing.T) {
		step := &Step{
			Name: "Checkout",
			Uses: "actions/checkout@v4",
		}

		if !step.IsAction() {
			t.Error("IsAction() should return true when Uses is set")
		}
	})

	t.Run("returns false when Uses is empty", func(t *testing.T) {
		step := &Step{
			Name: "Run tests",
			Run:  "go test ./...",
		}

		if step.IsAction() {
			t.Error("IsAction() should return false when Uses is empty")
		}
	})

	t.Run("returns true for composite action", func(t *testing.T) {
		step := &Step{
			Name: "Setup Go",
			Uses: "actions/setup-go@v5",
			With: map[string]string{
				"go-version": "1.21",
			},
		}

		if !step.IsAction() {
			t.Error("IsAction() should return true for action with With parameters")
		}
	})

	t.Run("returns false for step with only Run", func(t *testing.T) {
		step := &Step{
			Run: "echo hello",
		}

		if step.IsAction() {
			t.Error("IsAction() should return false for step with only Run")
		}
	})

	t.Run("returns true for local action", func(t *testing.T) {
		step := &Step{
			Uses: "./.github/actions/my-action",
		}

		if !step.IsAction() {
			t.Error("IsAction() should return true for local action")
		}
	})
}
