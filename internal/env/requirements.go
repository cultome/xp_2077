package env

import (
	"fmt"
	"os"
	"strings"
)

type RequirementStatus struct {
	Name    string
	Present bool
	Hint    string
}

type Report struct {
	Statuses []RequirementStatus
	Missing  bool
}

func Check(required []string) Report {
	statuses := make([]RequirementStatus, 0, len(required))
	missing := false

	for _, variable := range required {
		value := strings.TrimSpace(os.Getenv(variable))
		present := value != ""
		if !present {
			missing = true
		}
		statuses = append(statuses, RequirementStatus{
			Name:    variable,
			Present: present,
			Hint:    fmt.Sprintf("export %s=...", variable),
		})
	}
	return Report{Statuses: statuses, Missing: missing}
}
