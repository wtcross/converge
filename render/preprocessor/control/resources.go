package control

import (
	"fmt"
	"strings"

	"github.com/asteris-llc/converge/load/registry"
	"github.com/asteris-llc/converge/render/extensions"
	"github.com/asteris-llc/converge/resource"
	"github.com/pkg/errors"
)

// SwitchPreparer represents a switch resourc; the task it generates simply
// wraps the values and will not do anything during check or apply.
type SwitchPreparer struct {
	Branches []string `hcl:"branches"`
	cases    []*CasePreparer
}

func (s *SwitchPreparer) Cases() []*CasePreparer {
	return s.cases
}

func (s *SwitchPreparer) SetCases(cases []*CasePreparer) {
	s.cases = cases
}

func (s *SwitchPreparer) AppendCase(c *CasePreparer) {
	s.cases = append(s.cases, c)
}

// SwitchTask represents a resource.Task for a switch node.  It does not
// perform any operations and exists to provide structure to conditional
// evaluation in the graph and holds predicate state information.
type SwitchTask struct {
	Branches []string
}

type CasePreparer struct {
	Predicate string `hcl:"predicate"`
	Name      string `hcl:"name"`
	parent    *SwitchPreparer
}

// SetParent set's the parent of a case statement
func (c *CasePreparer) SetParent(s *SwitchPreparer) {
	c.parent = s
}

// GetParent gets the parent of a case
func (c *CasePreparer) GetParent() *SwitchPreparer {
	return c.parent
}

type CaseTask struct {
	*CasePreparer
}

// Prepare does stuff
func (s *SwitchPreparer) Prepare(resource.Renderer) (resource.Task, error) {
	task := &SwitchTask{Branches: s.Branches}
	for _, caseObj := range s.cases {
		if caseObj != nil {
			caseObj.SetParent(s)
		}
	}
	return task, nil
}

// Check does stuff
func (s *SwitchTask) Check(resource.Renderer) (resource.TaskStatus, error) {
	return &resource.Status{}, nil
}

// Apply does stuff
func (s *SwitchTask) Apply() (resource.TaskStatus, error) {
	return &resource.Status{}, nil
}

// Prepare does stuff
func (c *CasePreparer) Prepare(r resource.Renderer) (resource.Task, error) {
	predicate, err := r.Render("predicate", c.Predicate)
	if err != nil {
		return nil, err
	}

	c.Predicate = predicate
	return &CaseTask{c}, nil
}

// ShouldEvaluate returns true if the case has a valid parent and it is the
// selected branch for that parent
func (c *CasePreparer) ShouldEvaluate() bool {
	if c.parent == nil {
		return false
	}
	for _, br := range c.parent.Branches {
		if c.Name == br {
			t, _ := c.IsTrue()
			return t
		}
	}
	return false
}

// IsTrue returns true if the template precicate evaluates to "true", or "t",
// false if it returns "false", or "f", or if the pointer is nil, and returns
// false with an error otherwise.
func (c *CasePreparer) IsTrue() (bool, error) {
	if c == nil {
		return false, nil
	}
	return EvaluatePredicate(c.Predicate)
}

// EvaluatePredicate looks at a templated string and returns true if template
// execution results in the string "true" or t", and false if the string is
// "false" or "f".  In any other case an error is returned.
func EvaluatePredicate(predicate string) (bool, error) {
	lang := extensions.DefaultLanguage()
	if predicate == "" {
		return false, BadPredicate(predicate)
	}
	template := "{{ " + predicate + " }}"
	result, err := lang.Render(
		struct{}{},
		"predicate evaluation",
		template,
	)
	if err != nil {
		return false, errors.Wrap(err, "case evaluation failed")
	}

	truthiness := strings.TrimSpace(strings.ToLower(result.String()))

	switch truthiness {
	case "true", "t":
		return true, nil
	case "false", "f":
		return false, nil
	}
	return false, fmt.Errorf("%s: not a valid truth value; should be one of [f false t true]", truthiness)
}

// Check does stuff
func (c *CaseTask) Check(resource.Renderer) (resource.TaskStatus, error) {
	return &resource.Status{}, nil
}

// Apply does stuff
func (c *CaseTask) Apply() (resource.TaskStatus, error) {
	return &resource.Status{}, nil
}

// ConditionalTask represents a task that may or may not be executed. It's
// evaluation is determined by it's parent control-structure predicate.
type ConditionalTask struct {
	resource.Task
	*ConditionalPreparer
}

// EvaluationController represents an interface for a thing that can control
// conditional execution (e.g. a CasePreparer or CaseTask)
type EvaluationController interface {
	ShouldEvaluate() bool
}

// ConditionalPreparer wraps a preparer resource so thta a conditional task can
// be generated.
type ConditionalPreparer struct {
	resource.Resource
	controller EvaluationController
	Name       string
}

// SetExecutionController sets the private execution controller
func (c *ConditionalPreparer) SetExecutionController(ctrl EvaluationController) {
	c.controller = ctrl
}

// GetTask will return the task if it should be evaluated, and a nop-task
// otherwise.  The nop task will embed the original task so fields will still be
// resolvable.
func (c *ConditionalTask) GetTask() (resource.Task, bool) {
	if c.controller.ShouldEvaluate() {
		return c.Task, true
	}
	return &NopTask{c.Task}, true
}

// Apply will conditionally apply a task
func (c *ConditionalTask) Apply() (resource.TaskStatus, error) {
	if c.controller.ShouldEvaluate() {
		return c.Task.Apply()
	}
	return &resource.Status{}, nil
}

// Check will conditionally check a task
func (c *ConditionalTask) Check(r resource.Renderer) (resource.TaskStatus, error) {
	if c == nil {
		return &resource.Status{}, errors.New("conditional task is nil")
	}
	if c.controller.ShouldEvaluate() {
		return c.Task.Check(r)
	}
	return &resource.Status{}, nil
}

// Prepare returns a conditional task after preparing the underlying resource
func (c *ConditionalPreparer) Prepare(r resource.Renderer) (resource.Task, error) {
	if c == nil {
		return &ConditionalTask{}, errors.New("cannot create a conditional task from a nil preparer")
	}
	prepared, err := c.Resource.Prepare(r)
	if err != nil {
		return nil, err
	}
	return &ConditionalTask{
		Task:                prepared,
		ConditionalPreparer: c,
	}, nil
}

// NopTask is a task with accessible fields that will never execute
type NopTask struct {
	resource.Task
}

// Check is a NOP
func (n *NopTask) Check(resource.Renderer) (resource.TaskStatus, error) {
	msg := "Check: pruned branch not executing task"
	return &resource.Status{Output: []string{msg}}, nil
}

// Apply is a NOP
func (n *NopTask) Apply() (resource.TaskStatus, error) {
	msg := "Apply: pruned branch not executing task"
	return &resource.Status{Output: []string{msg}}, nil
}

func init() {
	registry.Register("macro.switch", (*SwitchPreparer)(nil), (*SwitchTask)(nil))
	registry.Register("macro.case", (*CasePreparer)(nil), (*CaseTask)(nil))
}