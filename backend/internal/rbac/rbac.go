package rbac

import gorbac "github.com/mikespook/gorbac/v3"

// Permission IDs
const (
	PermDSNRead    = "dsn:read"
	PermDSNWrite   = "dsn:write"
	PermDSNDelete  = "dsn:delete"
	PermDSNTest    = "dsn:test"
	PermUserRead   = "user:read"
	PermUserWrite  = "user:write"
	PermUserDelete = "user:delete"
	PermTeamRead   = "team:read"
	PermTeamWrite  = "team:write"
	PermTeamDelete = "team:delete"
)

var allPermissions = []string{
	PermDSNRead, PermDSNWrite, PermDSNDelete, PermDSNTest,
	PermUserRead, PermUserWrite, PermUserDelete,
	PermTeamRead, PermTeamWrite, PermTeamDelete,
}

var memberPermissions = []string{
	PermDSNRead, PermDSNTest,
	PermTeamRead,
}

// New creates and returns a configured RBAC instance with admin and member roles.
func New() *gorbac.RBAC[string] {
	r := gorbac.New[string]()

	admin := gorbac.NewRole("admin")
	for _, p := range allPermissions {
		admin.Assign(gorbac.NewPermission(p))
	}

	member := gorbac.NewRole("member")
	for _, p := range memberPermissions {
		member.Assign(gorbac.NewPermission(p))
	}

	r.Add(admin)
	r.Add(member)
	return r
}

// IsGranted checks whether the given role has the given permission.
func IsGranted(r *gorbac.RBAC[string], role, permission string) bool {
	return r.IsGranted(role, gorbac.NewPermission(permission), nil)
}
