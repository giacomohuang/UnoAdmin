package domain_test

import (
	"reflect"
	"testing"

	"github.com/unoadmin/admin-service/internal/domain"
)

func TestEffectivePermissionsMergesAndDeduplicates(t *testing.T) {
	roles := []domain.Role{
		{PermissionCodes: []string{"user:read", "role:read"}},
		{PermissionCodes: []string{"role:read", "tenant:read"}},
	}

	got := domain.EffectivePermissions(roles)
	want := []string{"user:read", "role:read", "tenant:read"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EffectivePermissions() = %v, want %v", got, want)
	}
}

func TestEffectivePermissionsCollapsesWildcard(t *testing.T) {
	roles := []domain.Role{
		{PermissionCodes: []string{"demo:read"}},
		{PermissionCodes: []string{"*"}},
	}

	got := domain.EffectivePermissions(roles)
	if !reflect.DeepEqual(got, []string{"*"}) {
		t.Fatalf("EffectivePermissions() = %v, want [ * ]", got)
	}
}

func TestCanMatchesFrontendSessionBehavior(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		code        string
		want        bool
	}{
		{name: "empty code is always allowed", permissions: nil, code: "", want: true},
		{name: "wildcard allows everything", permissions: []string{"*"}, code: "user:write", want: true},
		{name: "exact match", permissions: []string{"user:read"}, code: "user:read", want: true},
		{name: "missing code is denied", permissions: []string{"user:read"}, code: "user:write", want: false},
		{name: "no permissions denies", permissions: nil, code: "demo:read", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.Can(tt.permissions, tt.code); got != tt.want {
				t.Fatalf("Can(%v, %q) = %v, want %v", tt.permissions, tt.code, got, tt.want)
			}
		})
	}
}

func TestBuildPermissionTreePreservesOrderAndNesting(t *testing.T) {
	nodes := []domain.Permission{
		{ID: "user-write", Code: "user:write", ParentID: "user", SortOrder: 1},
		{ID: "system", Code: "system", SortOrder: 0},
		{ID: "user", Code: "user:read", ParentID: "system", SortOrder: 0},
	}

	tree := domain.BuildPermissionTree(nodes)
	if len(tree) != 1 || tree[0].Code != "system" {
		t.Fatalf("root = %+v, want single system root", tree)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Code != "user:read" {
		t.Fatalf("children = %+v, want user:read", tree[0].Children)
	}
	children := tree[0].Children[0].Children
	if len(children) != 1 || children[0].Code != "user:write" {
		t.Fatalf("grandchildren = %+v, want user:write", children)
	}
}

func TestListQueryNormalizeClampsPagination(t *testing.T) {
	got := domain.ListQuery{Page: 0, PageSize: 1000}.Normalize()
	if got.Page != 1 || got.PageSize != 100 {
		t.Fatalf("Normalize() = %+v, want page=1 pageSize=100", got)
	}
	if got.Offset() != 0 || got.Limit() != 100 {
		t.Fatalf("Offset/Limit = %d/%d, want 0/100", got.Offset(), got.Limit())
	}
}
