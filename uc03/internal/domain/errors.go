package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Error codes. A1 codes are returned to the Developer before any side effect;
// A2 codes describe a failed background execution.
const (
	CodeInvalidImage             = "INVALID_IMAGE_VERSION"
	CodeMissingConfiguration     = "MISSING_REQUIRED_CONFIGURATION"
	CodeConfigurationMismatch    = "CONFIGURATION_VERSION_MISMATCH"
	CodePartialVersionMismatch   = "PARTIAL_DEPLOYMENT_VERSION_MISMATCH"
	CodePartialCatalogMismatch   = "PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH"
	CodeRunningVersionAmbiguous  = "RUNNING_VERSION_NOT_UNIFORM"
	CodeDependencyCycle          = "DEPENDENCY_CYCLE"
	CodeDependencyUnresolved     = "DEPENDENCY_UNRESOLVED"
	CodeDependencyNotHealthy     = "DEPENDENCY_NOT_RUNNING_HEALTHY"
	CodeNoResourceDefinition     = "NO_MATCHING_RESOURCE_DEFINITION"
	CodeAmbiguousDefinition      = "AMBIGUOUS_RESOURCE_DEFINITION"
	CodeDefinitionModeChanged    = "RESOURCE_DEFINITION_MODE_CHANGED"
	CodeDefinitionChanged        = "RESOURCE_DEFINITION_CHANGED"
	CodeInvalidOutputReference   = "INVALID_OUTPUT_REFERENCE"
	CodeInvalidOverride          = "INVALID_OVERRIDE"
	CodeImmutableParameter       = "IMMUTABLE_PARAMETER_CHANGED"
	CodeUnsupportedTarget        = "UNSUPPORTED_TARGET_OR_CONTEXT"
	CodeDeploymentInProgress     = "DEPLOYMENT_IN_PROGRESS"
	CodeAlreadyConfirmed         = "DEPLOYMENT_ALREADY_CONFIRMED"
	CodePlanChanged              = "PLAN_CHANGED"
	CodeInvalidInput             = "INVALID_INPUT"
	CodeNotFound                 = "NOT_FOUND"
	CodeNothingToTeardown        = "NOTHING_TO_TEARDOWN"
	CodePlanChangedBeforeExecute = "PLAN_CHANGED_BEFORE_EXECUTION"
)

// ValidationError is an A1 rejection. It carries every problem found so the
// Developer can fix them in one pass.
type ValidationError struct {
	Problems []Problem
}

type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.Code + ": " + p.Message
	}
	return strings.Join(parts, "; ")
}

func (e *ValidationError) Add(code, format string, args ...any) {
	e.Problems = append(e.Problems, Problem{Code: code, Message: fmt.Sprintf(format, args...)})
}

// OrNil returns the error only when at least one problem was recorded.
func (e *ValidationError) OrNil() error {
	if e == nil || len(e.Problems) == 0 {
		return nil
	}
	return e
}

func Reject(code, format string, args ...any) error {
	v := &ValidationError{}
	v.Add(code, format, args...)
	return v
}

// HasCode reports whether err is a ValidationError containing code.
func HasCode(err error, code string) bool {
	var v *ValidationError
	if !errors.As(err, &v) {
		return false
	}
	for _, p := range v.Problems {
		if p.Code == code {
			return true
		}
	}
	return false
}

// ExecutionError is an A2 failure: which wave, component and step failed.
type ExecutionError struct {
	Wave      int
	Component string
	Step      StepName
	Err       error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("wave %d, %s, step %s: %v", e.Wave, e.Component, e.Step, e.Err)
}

func (e *ExecutionError) Unwrap() error { return e.Err }
