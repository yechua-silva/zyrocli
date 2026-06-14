package handoff

import (
	"errors"
	"fmt"
)

// Validate checks that all required fields are present in the payload.
func Validate(p *Payload) error {
	var errs []error

	if p.Version != "2.0" {
		errs = append(errs, fmt.Errorf("version: must be \"2.0\", got %q", p.Version))
	}

	if p.Source.System == "" {
		errs = append(errs, errors.New("source.system: required field is missing"))
	}

	if p.Project.Name == "" {
		errs = append(errs, errors.New("project.name: required field is missing"))
	}

	if p.Project.Language == "" {
		errs = append(errs, errors.New("project.language: required field is missing"))
	}

	if p.Governance.Mode == "" {
		errs = append(errs, errors.New("governance.mode: required field is missing"))
	}

	if p.Testing.Strategy == "" {
		errs = append(errs, errors.New("testing.strategy: required field is missing"))
	}

	return errors.Join(errs...)
}
