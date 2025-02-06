package main

type Quark struct {
	Action   func() (interface{}, error) // Lambda function with no parameters that returns an error
	Rollback func(interface{})           // Lambda function with one parameter of type interface{}
}

// NewQuark function creates a new Quark instance
// If no rollback function is provided, a default no-op rollback function is used
func NewQuark(action func() (interface{}, error), rollback func(interface{})) *Quark {
	if rollback == nil {
		// Provide a default no-op Rollback function
		rollback = func(data interface{}) {}
	}

	return &Quark{
		Action:   action,
		Rollback: rollback,
	}
}

type Atom struct {
	quarks       []*Quark
	intermediate []any
}

func NewAtom(quarks ...*Quark) *Atom {
	return &Atom{
		quarks:       quarks,
		intermediate: make([]any, 0),
	}
}

func (a *Atom) Run() error {
	// perform actions sequentially
	for i, q := range a.quarks {
		res, err := q.Action()
		if err != nil {
			// if one fails, rollback all previous completed actions
			for j := 0; j < i; j++ {
				a.quarks[j].Rollback(a.intermediate[j])
			}
			return err
		}
		a.intermediate = append(a.intermediate, res)
	}
	return nil
}
