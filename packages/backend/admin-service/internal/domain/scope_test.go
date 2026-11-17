package domain_test

import (
	"reflect"
	"testing"

	"github.com/unoadmin/admin-service/internal/domain"
)

func TestValidateTenantModeMatchesCustomerShape(t *testing.T) {
	cases := []struct {
		name     string
		mode     string
		entities int
		wantErr  bool
	}{
		{"standalone with one entity", domain.TenantModeStandalone, 1, false},
		{"standalone without entity", domain.TenantModeStandalone, 0, false},
		{"standalone with two entities", domain.TenantModeStandalone, 2, true},
		{"group with many entities", domain.TenantModeGroup, 6, false},
		{"unknown mode", "reseller", 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateTenantMode(tc.mode, tc.entities)
			if tc.wantErr && err == nil {
				t.Fatal("ValidateTenantMode() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateTenantMode() error = %v", err)
			}
		})
	}
}

func TestVisibleEntitiesFollowsDataScope(t *testing.T) {
	entities := []domain.LegalEntity{
		{ID: "le-hq", Type: domain.LegalEntityTypeGroup},
		{ID: "le-east", Type: domain.LegalEntityTypeCompany},
		{ID: "le-south", Type: domain.LegalEntityTypeCompany},
	}

	tenantScope := &domain.Principal{DataScope: domain.DataScopeTenant}
	if got := domain.VisibleEntities(tenantScope, entities); len(got) != 3 {
		t.Fatalf("tenant scope visible = %d, want 3", len(got))
	}

	entityScope := &domain.Principal{DataScope: domain.DataScopeEntity, LegalEntityIDs: []string{"le-east"}}
	got := domain.VisibleEntities(entityScope, entities)
	if len(got) != 1 || got[0].ID != "le-east" {
		t.Fatalf("entity scope visible = %+v, want only le-east", got)
	}

	platform := &domain.Principal{PlatformAdmin: true, DataScope: domain.DataScopeEntity}
	if got := domain.VisibleEntities(platform, entities); len(got) != 3 {
		t.Fatalf("platform admin visible = %d, want 3", len(got))
	}
}

func TestWidenScopeKeepsTheBroadestRange(t *testing.T) {
	if got := domain.WidenScope(domain.DataScopeEntity, domain.DataScopeTenant); got != domain.DataScopeTenant {
		t.Fatalf("WidenScope(entity, tenant) = %q, want tenant", got)
	}
	if got := domain.WidenScope(domain.DataScopeTenant, domain.DataScopeSelf); got != domain.DataScopeTenant {
		t.Fatalf("WidenScope(tenant, self) = %q, want tenant", got)
	}
	if got := domain.WidenScope(domain.DataScopeSelf, domain.DataScopeEntity); got != domain.DataScopeEntity {
		t.Fatalf("WidenScope(self, entity) = %q, want entity", got)
	}
}

func TestBuildOrgTreeSortsAndNests(t *testing.T) {
	nodes := []domain.OrgUnit{
		{ID: "org-sales", ParentID: "org-root", Name: "销售部", Sort: 1},
		{ID: "org-root", Name: "华东公司", Sort: 0},
		{ID: "org-service", ParentID: "org-root", Name: "售后服务部", Sort: 0},
	}
	tree := domain.BuildOrgTree(nodes)
	if len(tree) != 1 || tree[0].ID != "org-root" {
		t.Fatalf("tree roots = %+v, want org-root", tree)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("children = %+v, want two departments", tree[0].Children)
	}
	if tree[0].Children[0].ID != "org-service" || tree[0].Children[1].ID != "org-sales" {
		t.Fatalf("children order = %+v, want sort ascending", tree[0].Children)
	}
}

func TestSortLegalEntitiesPutsGroupFirst(t *testing.T) {
	entities := []domain.LegalEntity{
		{ID: "le-b", Type: domain.LegalEntityTypeCompany},
		{ID: "le-a", Type: domain.LegalEntityTypeGroup},
	}
	got := domain.SortLegalEntities(entities)
	if !reflect.DeepEqual([]string{got[0].ID, got[1].ID}, []string{"le-a", "le-b"}) {
		t.Fatalf("sorted = %+v, want group entity first", got)
	}
}
