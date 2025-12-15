package rbac

type Role struct {
	Name      string
	Resources []string
	Verbs     []Action
}

type Subject struct {
	Kind string
	Name string
}

type RoleBinding struct {
	Name     string
	Subjects []Subject
	RoleName string
	Scopes   []string
}
