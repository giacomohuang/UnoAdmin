package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/unoadmin/admin-service/internal/admin"
	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/testutil"
)

// fixtures 构造覆盖集团与单体两类租户的管理服务装配。
func fixtures() (*admin.Service, *testutil.TenantRepo, *testutil.LegalEntityRepo) {
	tenants := testutil.NewTenantRepo(
		domain.Tenant{ID: "tenant-holding", Name: "汉唐控股集团", Code: "HANTANG", Mode: domain.TenantModeGroup, Status: domain.TenantActive},
		domain.Tenant{ID: "tenant-mingyuan", Name: "明远科技", Code: "MINGYUAN", Mode: domain.TenantModeStandalone, Status: domain.TenantActive},
	)
	entities := testutil.NewLegalEntityRepo(
		domain.LegalEntity{ID: "le-hq", TenantID: "tenant-holding", Type: domain.LegalEntityTypeGroup, Name: "集团总部", Code: "HT-HQ", Status: domain.TenantActive},
		domain.LegalEntity{ID: "le-east", TenantID: "tenant-holding", ParentID: "le-hq", Type: domain.LegalEntityTypeCompany, Name: "华东公司", Code: "HT-EAST", Status: domain.TenantActive},
		domain.LegalEntity{ID: "le-mingyuan", TenantID: "tenant-mingyuan", Type: domain.LegalEntityTypeCompany, Name: "明远科技", Code: "MY", Status: domain.TenantActive},
	)
	accounts := testutil.NewAccountRepo(
		domain.Account{ID: "acc-group", Username: "chen.yu", DisplayName: "陈宇", Status: domain.UserActive, TenantID: "tenant-holding", DataScope: domain.DataScopeTenant, RoleIDs: []string{"role-group"}, LegalEntityIDs: []string{"le-hq"}},
		domain.Account{ID: "acc-east", Username: "zhao.ming", DisplayName: "赵明", Status: domain.UserActive, TenantID: "tenant-holding", DataScope: domain.DataScopeEntity, RoleIDs: []string{"role-entity"}, LegalEntityIDs: []string{"le-east"}},
		domain.Account{ID: "acc-solo", Username: "li.na", DisplayName: "李娜", Status: domain.UserActive, TenantID: "tenant-mingyuan", DataScope: domain.DataScopeTenant},
	)
	roles := testutil.NewRoleRepo(
		domain.Role{ID: "role-group", TenantID: "tenant-holding", Name: "集团管理员", Code: "GROUP_ADMIN", Status: domain.TenantActive, DataScope: domain.DataScopeTenant, PermissionCodes: []string{"account:view"}},
		domain.Role{ID: "role-entity", TenantID: "tenant-holding", Name: "主体管理员", Code: "ENTITY_ADMIN", Status: domain.TenantActive, DataScope: domain.DataScopeEntity, PermissionCodes: []string{"org:view"}},
	)
	permissions := testutil.NewPermissionRepo(
		domain.Permission{ID: "p-account", Name: "账户管理", Code: "account:view", Type: "page", Scope: domain.PermissionScopeTenant},
		domain.Permission{ID: "p-org", Name: "组织架构", Code: "org:view", Type: "page", Scope: domain.PermissionScopeEntity},
	)
	orgs := testutil.NewOrgRepo(
		domain.OrgUnit{ID: "org-east", TenantID: "tenant-holding", LegalEntityID: "le-east", Name: "华东公司", Path: []string{"org-east"}, Status: domain.TenantActive},
	)
	svc := admin.NewService(accounts, roles, permissions, tenants, entities, orgs, testutil.PlainHasher{})
	return svc, tenants, entities
}

// groupPrincipal 返回集团管理员的调用主体（租户级数据范围）。
func groupPrincipal() *domain.Principal {
	return &domain.Principal{
		AccountID: "acc-group", TenantID: "tenant-holding", LegalEntityID: "le-hq",
		DataScope: domain.DataScopeTenant, LegalEntityIDs: []string{"le-hq"},
		Permissions: []string{"*"},
	}
}

// entityPrincipal 返回主体管理员的调用主体（主体级数据范围）。
func entityPrincipal() *domain.Principal {
	return &domain.Principal{
		AccountID: "acc-east", TenantID: "tenant-holding", LegalEntityID: "le-east",
		DataScope: domain.DataScopeEntity, LegalEntityIDs: []string{"le-east"},
		Permissions: []string{"*"},
	}
}

// platformPrincipal 返回平台运营管理员的调用主体。
func platformPrincipal() *domain.Principal {
	return &domain.Principal{
		AccountID: "acc-platform", DataScope: domain.DataScopeTenant,
		PlatformAdmin: true, Permissions: []string{"*"},
	}
}

func TestListAccountsFillsRoleAndEntityNames(t *testing.T) {
	svc, _, _ := fixtures()

	page, err := svc.ListAccounts(context.Background(), groupPrincipal(), domain.ListQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAccounts() error = %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("total = %d, want only the two accounts in the tenant", page.Total)
	}
	var group domain.Account
	for _, account := range page.Items {
		if account.ID == "acc-group" {
			group = account
		}
	}
	if len(group.RoleNames) != 1 || group.RoleNames[0] != "集团管理员" {
		t.Fatalf("roleNames = %v, want 集团管理员", group.RoleNames)
	}
	if len(group.LegalEntityNames) != 1 || group.LegalEntityNames[0] != "集团总部" {
		t.Fatalf("legalEntityNames = %v, want 集团总部", group.LegalEntityNames)
	}
}

func TestListLegalEntitiesRespectsDataScope(t *testing.T) {
	svc, _, _ := fixtures()

	groupItems, err := svc.ListLegalEntities(context.Background(), groupPrincipal(), domain.ListQuery{})
	if err != nil {
		t.Fatalf("ListLegalEntities(group) error = %v", err)
	}
	if len(groupItems) != 2 {
		t.Fatalf("group visible entities = %d, want 2", len(groupItems))
	}
	entityItems, err := svc.ListLegalEntities(context.Background(), entityPrincipal(), domain.ListQuery{})
	if err != nil {
		t.Fatalf("ListLegalEntities(entity) error = %v", err)
	}
	if len(entityItems) != 1 || entityItems[0].ID != "le-east" {
		t.Fatalf("entity visible entities = %+v, want only le-east", entityItems)
	}
}

func TestSaveTenantCreatesSingleEntityForStandalone(t *testing.T) {
	svc, tenants, entities := fixtures()
	platform := &domain.Principal{PlatformAdmin: true, DataScope: domain.DataScopeTenant, Permissions: []string{"*"}}

	overview, err := svc.SaveTenant(context.Background(), platform, admin.TenantInput{
		Name: "甲方独资有限公司", Code: "party-a", Mode: domain.TenantModeStandalone, Contact: "财务部",
	})
	if err != nil {
		t.Fatalf("SaveTenant() error = %v", err)
	}
	if overview.Mode != domain.TenantModeStandalone || overview.EntityCount != 1 {
		t.Fatalf("overview = %+v, want standalone tenant with one entity", overview)
	}
	if _, err := tenants.Get(context.Background(), overview.ID); err != nil {
		t.Fatalf("created tenant not persisted: %v", err)
	}
	created, err := entities.List(context.Background(), domain.ListQuery{TenantID: overview.ID})
	if err != nil {
		t.Fatalf("list entities error = %v", err)
	}
	if len(created) != 1 || created[0].Type != domain.LegalEntityTypeCompany {
		t.Fatalf("entities = %+v, want one company entity", created)
	}
}

func TestSaveTenantRejectsNonPlatformCaller(t *testing.T) {
	svc, _, _ := fixtures()

	if _, err := svc.SaveTenant(context.Background(), groupPrincipal(), admin.TenantInput{
		Name: "越权租户", Code: "X", Mode: domain.TenantModeGroup,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SaveTenant() error = %v, want ErrForbidden", err)
	}
}

func TestSaveLegalEntityEnforcesGroupRules(t *testing.T) {
	svc, _, _ := fixtures()
	principal := groupPrincipal()

	// 集团租户只允许一个集团总部主体。
	if _, err := svc.SaveLegalEntity(context.Background(), principal, admin.LegalEntityInput{
		TenantID: "tenant-holding", Type: domain.LegalEntityTypeGroup, Name: "第二总部", Code: "HQ2",
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate group entity error = %v, want ErrConflict", err)
	}

	created, err := svc.SaveLegalEntity(context.Background(), principal, admin.LegalEntityInput{
		TenantID: "tenant-holding", ParentID: "le-hq", Type: domain.LegalEntityTypeCompany, Name: "华北公司", Code: "HT-NORTH",
	})
	if err != nil {
		t.Fatalf("SaveLegalEntity() error = %v", err)
	}
	if created.ParentID != "le-hq" || created.TenantID != "tenant-holding" {
		t.Fatalf("created = %+v, want company under le-hq", created)
	}

	// 经营公司必须挂在集团主体下，指向公司主体应被拒绝。
	if _, err := svc.SaveLegalEntity(context.Background(), principal, admin.LegalEntityInput{
		TenantID: "tenant-holding", ParentID: "le-east", Type: domain.LegalEntityTypeCompany, Name: "孙公司", Code: "HT-SUB",
	}); err == nil {
		t.Fatal("SaveLegalEntity() should reject a non-group parent")
	}
}

func TestSaveLegalEntityRejectsSecondEntityInStandaloneTenant(t *testing.T) {
	svc, _, _ := fixtures()
	platform := &domain.Principal{PlatformAdmin: true, TenantID: "tenant-mingyuan", DataScope: domain.DataScopeTenant, Permissions: []string{"*"}}

	if _, err := svc.SaveLegalEntity(context.Background(), platform, admin.LegalEntityInput{
		TenantID: "tenant-mingyuan", Type: domain.LegalEntityTypeCompany, Name: "第二主体", Code: "MY2",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("standalone second entity error = %v, want ErrInvalid", err)
	}
}

func TestSaveOrgMaintainsPathAndTree(t *testing.T) {
	svc, _, _ := fixtures()

	child, err := svc.SaveOrg(context.Background(), entityPrincipal(), admin.OrgInput{
		LegalEntityID: "le-east", ParentID: "org-east", Name: "销售部", Type: "department",
	})
	if err != nil {
		t.Fatalf("SaveOrg() error = %v", err)
	}
	if len(child.Path) != 2 || child.Path[0] != "org-east" || child.Path[1] != child.ID {
		t.Fatalf("path = %v, want [org-east %s]", child.Path, child.ID)
	}

	tree, err := svc.ListOrgs(context.Background(), entityPrincipal(), "le-east")
	if err != nil {
		t.Fatalf("ListOrgs() error = %v", err)
	}
	if len(tree) != 1 || len(tree[0].Children) != 1 {
		t.Fatalf("tree = %+v, want root with one child", tree)
	}
}

func TestSaveAccountRejectsForeignLegalEntity(t *testing.T) {
	svc, _, _ := fixtures()

	// 集团管理员不能把账户授权到其它租户的主体。
	if _, err := svc.SaveAccount(context.Background(), groupPrincipal(), admin.AccountInput{
		Username: "new.user", DisplayName: "新同事", Password: "password123",
		DataScope: domain.DataScopeEntity, LegalEntityIDs: []string{"le-mingyuan"},
	}); err == nil {
		t.Fatal("SaveAccount() should reject an entity from another tenant")
	}
}

func TestListAccountsHidesAccountsOutsideEntityScope(t *testing.T) {
	svc, _, _ := fixtures()

	page, err := svc.ListAccounts(context.Background(), entityPrincipal(), domain.ListQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAccounts() error = %v", err)
	}
	if page.Total != 1 || page.Items[0].ID != "acc-east" {
		t.Fatalf("accounts = %+v, want only the account sharing the authorized entity", page.Items)
	}
}

func TestEntityOperatorCannotGrantForeignEntity(t *testing.T) {
	svc, _, _ := fixtures()

	if _, err := svc.SaveAccount(context.Background(), entityPrincipal(), admin.AccountInput{
		Username: "new.user", DisplayName: "新同事", Password: "password123",
		DataScope: domain.DataScopeEntity, LegalEntityIDs: []string{"le-hq"},
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SaveAccount(foreign entity) error = %v, want ErrForbidden", err)
	}
	if _, err := svc.SaveAccount(context.Background(), entityPrincipal(), admin.AccountInput{
		Username: "new.user", DisplayName: "新同事", Password: "password123",
		DataScope: domain.DataScopeTenant, LegalEntityIDs: []string{"le-east"},
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SaveAccount(tenant scope) error = %v, want ErrForbidden", err)
	}
}

func TestSavePermissionRequiresPlatformAdmin(t *testing.T) {
	svc, _, _ := fixtures()

	if _, err := svc.SavePermission(context.Background(), groupPrincipal(), admin.PermissionInput{
		Name: "导出账户", Code: "account:export", Type: "action", Scope: domain.PermissionScopeTenant, ParentID: "p-account",
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SavePermission() error = %v, want ErrForbidden", err)
	}
}

func TestSavePermissionRejectsDuplicateCodeAndCycle(t *testing.T) {
	svc, _, _ := fixtures()
	principal := platformPrincipal()

	if _, err := svc.SavePermission(context.Background(), principal, admin.PermissionInput{
		Name: "重复权限", Code: "account:view", Type: "page", Scope: domain.PermissionScopeTenant,
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate code error = %v, want ErrConflict", err)
	}

	// 让 p-org 挂到 p-account 下再尝试反向移动，形成父子环。
	created, err := svc.SavePermission(context.Background(), principal, admin.PermissionInput{
		Name: "组织详情", Code: "org:detail", Type: "page", Scope: domain.PermissionScopeEntity, ParentID: "p-org",
	})
	if err != nil {
		t.Fatalf("create permission error = %v", err)
	}
	if _, err := svc.SavePermission(context.Background(), principal, admin.PermissionInput{
		ID: "p-org", Name: "组织架构", Code: "org:view", Type: "page", Scope: domain.PermissionScopeEntity, ParentID: created.ID,
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("cycle error = %v, want ErrInvalid", err)
	}
}

func TestReorderPermissionsRewritesSiblingOrder(t *testing.T) {
	svc, _, _ := fixtures()

	tree, err := svc.ReorderPermissions(context.Background(), platformPrincipal(), admin.PermissionReorderInput{
		ParentID: "", IDs: []string{"p-org", "p-account"},
	})
	if err != nil {
		t.Fatalf("ReorderPermissions() error = %v", err)
	}
	if len(tree) != 2 || tree[0].ID != "p-org" || tree[1].ID != "p-account" {
		t.Fatalf("tree order = %+v, want p-org before p-account", tree)
	}

	// 跨父级提交视为非法拖拽。
	if _, err := svc.ReorderPermissions(context.Background(), platformPrincipal(), admin.PermissionReorderInput{
		ParentID: "p-account", IDs: []string{"p-org"},
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("cross-parent reorder error = %v, want ErrInvalid", err)
	}
}

func TestDeletePermissionRejectsParentWithChildren(t *testing.T) {
	svc, _, _ := fixtures()
	principal := platformPrincipal()

	// 夹具中的 p-org 没有子节点，先补一个再删除父节点。
	if _, err := svc.SavePermission(context.Background(), principal, admin.PermissionInput{
		Name: "组织详情", Code: "org:detail", Type: "page", Scope: domain.PermissionScopeEntity, ParentID: "p-org",
	}); err != nil {
		t.Fatalf("create permission error = %v", err)
	}
	if err := svc.DeletePermission(context.Background(), principal, "p-org"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("delete parent error = %v, want ErrConflict", err)
	}
}

func TestListRolesCountsAccounts(t *testing.T) {
	svc, _, _ := fixtures()

	page, err := svc.ListRoles(context.Background(), groupPrincipal(), domain.ListQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListRoles() error = %v", err)
	}
	counts := map[string]int{}
	for _, role := range page.Items {
		counts[role.ID] = role.UserCount
	}
	if counts["role-group"] != 1 || counts["role-entity"] != 1 {
		t.Fatalf("user counts = %+v, want one account per role", counts)
	}
}

func TestStatsScopedToCallerTenant(t *testing.T) {
	svc, _, _ := fixtures()

	stats, err := svc.Stats(context.Background(), groupPrincipal())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats["legalEntities"] != 2 || stats["accounts"] != 2 || stats["roles"] != 2 {
		t.Fatalf("stats = %+v, want 2 entities / 2 accounts / 2 roles", stats)
	}
}

func TestStatsForEntityScopeCountsOwnAccounts(t *testing.T) {
	svc, _, _ := fixtures()

	stats, err := svc.Stats(context.Background(), entityPrincipal())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats["legalEntities"] != 1 || stats["accounts"] != 1 {
		t.Fatalf("stats = %+v, want only the authorized entity and its account", stats)
	}
}
