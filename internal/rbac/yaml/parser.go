package yaml

import (
	"errors"
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/rbac"

	goYaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
)

func ParseYAML(d []byte) (
	tokens []rbac.Token,
	users []rbac.User,
	roles []rbac.Role,
	roleBindings []rbac.RoleBinding,
	err error,
) {
	file, err := parser.ParseBytes(d, 0)
	if err != nil {
		return nil, nil, nil, nil, errors.Join(ErrWhileParsing, err)
	}

	for _, doc := range file.Docs {
		docBytes := []byte(doc.String())

		var common CommonManifest
		if err = goYaml.Unmarshal(docBytes, &common); err != nil {
			return nil, nil, nil, nil, errors.Join(ErrWhileUnmarshal, err)
		}

		switch common.Kind {

		case "Token":
			var m TokenManifest
			if err = goYaml.Unmarshal(docBytes, &m); err != nil {
				return nil, nil, nil, nil, errors.Join(ErrWhileUnmarshal, err)
			}
			tokens = append(tokens, rbac.Token{
				Name:      common.Metadata.Name,
				Value:     m.Spec.Value,
				Username:  m.Spec.Username,
				ExpiresAt: m.Spec.ExpiresAt,
			})

		case "User":
			var m UserManifest
			if err = goYaml.Unmarshal(docBytes, &m); err != nil {
				return nil, nil, nil, nil, errors.Join(ErrWhileUnmarshal, err)
			}
			users = append(users, rbac.User{
				Name:         common.Metadata.Name,
				PasswordHash: m.Spec.PasswordHash,
				Groups:       m.Spec.Groups,
			})

		case "Role":
			var m RoleManifest
			if err = goYaml.Unmarshal(docBytes, &m); err != nil {
				return nil, nil, nil, nil, errors.Join(ErrWhileUnmarshal, err)
			}
			verbs, err := rbac.ParseVerbs(m.Spec.Verbs)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("%w for %s", err, m.Spec.Verbs)
			}
			roles = append(roles, rbac.Role{
				Name:      common.Metadata.Name,
				Resources: m.Spec.Resources,
				Verbs:     verbs,
			})

		case "RoleBinding":
			var m RoleBindingManifest
			if err = goYaml.Unmarshal(docBytes, &m); err != nil {
				return nil, nil, nil, nil, errors.Join(ErrWhileUnmarshal, err)
			}

			subjects := make([]rbac.Subject, 0, len(m.Spec.Subjects))
			for _, s := range m.Spec.Subjects {
				subjects = append(subjects, rbac.Subject{
					Kind: s.Kind,
					Name: s.Name,
				})
			}

			roleBindings = append(roleBindings, rbac.RoleBinding{
				Name:     common.Metadata.Name,
				RoleName: m.Spec.RoleRef.Name,
				Subjects: subjects,
				Scopes:   m.Spec.Scopes,
			})
		}
	}
	return tokens, users, roles, roleBindings, nil
}
