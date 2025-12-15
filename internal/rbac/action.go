package rbac

type Action string

const (
	ActionPull   Action = "pull"
	ActionPush   Action = "push"
	ActionDelete Action = "delete"
)

func ParseActions(s []string) ([]Action, error) {
	actions := []Action{}
	for _, action := range s {
		if action == "*" {
			actions = append(actions, ActionPull, ActionPush, ActionDelete)
			continue
		}

		switch Action(action) {
		case ActionPull, ActionPush, ActionDelete:
			actions = append(actions, Action(action))
		default:
			return nil, ErrActionInvalid
		}
	}
	return actions, nil
}
