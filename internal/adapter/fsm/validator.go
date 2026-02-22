package fsm

import (
	"context"
	"errors"

	loopfsm "github.com/looplab/fsm"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// Compile-time check: Validator implements domain.TransitionValidator.
var _ domain.TransitionValidator = (*Validator)(nil)

// events converts domain.Transitions into looplab/fsm EventDesc format.
// It consolidates transitions with the same event+destination into a single
// EventDesc with multiple source states (e.g., EventDelete from "active"
// and "suspended" both go to "deleting").
var events = buildEvents()

func buildEvents() []loopfsm.EventDesc {
	type key struct {
		event string
		dst   string
	}
	grouped := make(map[key][]string)
	order := make([]key, 0)

	for _, t := range domain.Transitions {
		k := key{event: string(t.Event), dst: string(t.Dst)}
		if _, exists := grouped[k]; !exists {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], string(t.Src))
	}

	out := make([]loopfsm.EventDesc, 0, len(order))
	for _, k := range order {
		out = append(out, loopfsm.EventDesc{
			Name: k.event,
			Src:  grouped[k],
			Dst:  k.dst,
		})
	}
	return out
}

// Validator implements domain.TransitionValidator using looplab/fsm.
// It creates a short-lived FSM instance per Apply call, initialized with
// the tenant's current state. This is necessary because looplab/fsm is
// stateful (it tracks the current state internally).
type Validator struct{}

// New creates a new FSM-backed transition validator.
func New() *Validator {
	return &Validator{}
}

// Apply checks if the given event is valid from the current status and
// returns the destination status. Returns a domain.TransitionError if
// the transition is not allowed.
func (v *Validator) Apply(ctx context.Context, current domain.Status, event domain.Event) (domain.Status, error) {
	machine := loopfsm.NewFSM(string(current), events, nil)

	if err := machine.Event(ctx, string(event)); err != nil {
		var invalidEvent loopfsm.InvalidEventError
		var noTransition loopfsm.NoTransitionError
		if errors.As(err, &invalidEvent) || errors.As(err, &noTransition) {
			return "", &domain.TransitionError{
				Event:   event,
				Current: current,
			}
		}
		return "", err
	}

	return domain.Status(machine.Current()), nil
}
