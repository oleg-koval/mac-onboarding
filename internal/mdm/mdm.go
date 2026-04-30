package mdm

import (
	"fmt"
	"os/exec"
	"strings"
)

// Status describes MDM enrollment state.
type Status struct {
	Enrolled bool
	Details  string
}

// Probe runs `profiles status -type enrollment` and returns enrollment state.
// Non-fatal: if the command is unavailable, returns Enrolled=false.
func Probe() Status {
	out, err := exec.Command("profiles", "status", "-type", "enrollment").CombinedOutput()
	if err != nil {
		// profiles may not exist on non-MDM Macs — that's fine.
		return Status{Enrolled: false}
	}
	s := string(out)
	enrolled := strings.Contains(s, "MDM enrollment: Yes") ||
		strings.Contains(s, "Enrolled via DEP: Yes")
	return Status{
		Enrolled: enrolled,
		Details:  strings.TrimSpace(s),
	}
}

// Warn prints an MDM warning if enrolled. Call before any restricted write.
func Warn(operation string) {
	if s := Probe(); s.Enrolled {
		fmt.Printf("  ⚠  MDM enrolled — %s may be restricted\n", operation)
	}
}
