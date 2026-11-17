// Package domain 定义与前端 core 对齐的多租户 SaaS 领域实体、仓储端口与权限规则。
//
// 多租户模型分三层：
//   - Tenant（租户）是采购、计费与数据隔离的最外层边界，分为集团型与单体型两种模式；
//   - LegalEntity（经营主体）是租户内部相互隔离的经营实体，集团租户下可有多个，
//     单体租户下有且只有一个；
//   - OrgUnit（组织节点）挂在经营主体之下，描述部门、门店、项目组等内部结构。
package domain

import "time"

// ListQuery 对应前端 core 的 ListQuery，并携带租户与经营主体范围过滤条件。
type ListQuery struct {
	Page          int
	PageSize      int
	Search        string
	TenantID      string
	LegalEntityID string
	// LegalEntityIDs 用于主体级账户过滤：命中任一授权主体即可见。
	LegalEntityIDs []string
	Status         string
}

// Normalize 将分页参数收敛到安全范围，避免负数或过大的页容量打到数据库。
func (q ListQuery) Normalize() ListQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}

// Offset 返回 MongoDB skip 的位置。
func (q ListQuery) Offset() int64 {
	q = q.Normalize()
	return int64((q.Page - 1) * q.PageSize)
}

// Limit 返回 MongoDB limit 的大小。
func (q ListQuery) Limit() int64 {
	return int64(q.Normalize().PageSize)
}

// PageResult 对应前端 core 的 PageResult<Item>。
type PageResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

// Tenant 是采购与数据隔离的最外层单位。
//
// Mode 决定客户形态：
//   - TenantModeGroup：集团统一采购，租户下挂多个独立经营的 LegalEntity；
//   - TenantModeStandalone：单体小公司采购，租户下有且只有一个 LegalEntity。
type Tenant struct {
	ID            string        `json:"id" bson:"_id"`
	Name          string        `json:"name" bson:"name"`
	Code          string        `json:"code" bson:"code"`
	Mode          string        `json:"mode" bson:"mode"`
	Plan          string        `json:"plan" bson:"plan"`
	Contact       string        `json:"contact" bson:"contact"`
	Status        string        `json:"status" bson:"status"`
	LegalEntities []LegalEntity `json:"legalEntities,omitempty" bson:"-"`
	EntityCount   int           `json:"entityCount" bson:"-"`
	AccountCount  int           `json:"accountCount" bson:"-"`
	CreatedAt     time.Time     `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt" bson:"updatedAt"`
}

// LegalEntity 是租户内部相互隔离的经营主体（法人）。
//
// Type 为 GROUP 时代表集团总部，可作为其它 COMPANY 主体的父级；
// Type 为 COMPANY 时代表独立核算的经营公司。
type LegalEntity struct {
	ID        string    `json:"id" bson:"_id"`
	TenantID  string    `json:"tenantId" bson:"tenantId"`
	ParentID  string    `json:"parentId" bson:"parentId"`
	Type      string    `json:"type" bson:"type"`
	Name      string    `json:"name" bson:"name"`
	Code      string    `json:"code" bson:"code"`
	Address   string    `json:"address" bson:"address"`
	Status    string    `json:"status" bson:"status"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// OrgUnit 是经营主体内部的组织节点，使用 Path 保存从根到自身的 ID 链。
type OrgUnit struct {
	ID            string    `json:"id" bson:"_id"`
	TenantID      string    `json:"tenantId" bson:"tenantId"`
	LegalEntityID string    `json:"legalEntityId" bson:"legalEntityId"`
	ParentID      string    `json:"parentId" bson:"parentId"`
	Name          string    `json:"name" bson:"name"`
	Type          string    `json:"type" bson:"type"`
	Path          []string  `json:"path" bson:"path"`
	Sort          int       `json:"sort" bson:"sort"`
	Status        string    `json:"status" bson:"status"`
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
	Children      []OrgUnit `json:"children,omitempty" bson:"-"`
}

// Account 是管理端登录账户，同时描述其租户、经营主体与组织授权范围。
//
// DataScope 决定可见数据边界：
//   - DataScopeTenant：可查看租户下全部经营主体（集团总部管理员）；
//   - DataScopeEntity：仅可见 LegalEntityIDs 中的经营主体；
//   - DataScopeSelf：仅可见本人所属组织及下级（部门负责人）。
type Account struct {
	ID               string    `json:"id" bson:"_id"`
	Username         string    `json:"username" bson:"username"`
	PasswordHash     string    `json:"-" bson:"passwordHash"`
	DisplayName      string    `json:"displayName" bson:"displayName"`
	Email            string    `json:"email" bson:"email"`
	Phone            string    `json:"phone" bson:"phone"`
	Status           string    `json:"status" bson:"status"`
	TenantID         string    `json:"tenantId" bson:"tenantId"`
	DataScope        string    `json:"dataScope" bson:"dataScope"`
	PlatformAdmin    bool      `json:"platformAdmin" bson:"platformAdmin"`
	LegalEntityIDs   []string  `json:"legalEntityIds" bson:"legalEntityIds"`
	OrgIDs           []string  `json:"orgIds" bson:"orgIds"`
	RoleIDs          []string  `json:"roleIds" bson:"roleIds"`
	RoleNames        []string  `json:"roleNames" bson:"-"`
	LegalEntityNames []string  `json:"legalEntityNames" bson:"-"`
	CreatedAt        time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt" bson:"updatedAt"`
}

// Public 返回可安全下发到前端的账户副本，清除密码哈希并补齐空数组。
func (a Account) Public() Account {
	a.PasswordHash = ""
	if a.RoleIDs == nil {
		a.RoleIDs = []string{}
	}
	if a.RoleNames == nil {
		a.RoleNames = []string{}
	}
	if a.LegalEntityIDs == nil {
		a.LegalEntityIDs = []string{}
	}
	if a.OrgIDs == nil {
		a.OrgIDs = []string{}
	}
	if a.LegalEntityNames == nil {
		a.LegalEntityNames = []string{}
	}
	return a
}

// Role 是租户内的权限集合。
//
// LegalEntityID 为空表示租户级角色（可跨主体授予），否则为该主体专属角色。
type Role struct {
	ID              string    `json:"id" bson:"_id"`
	TenantID        string    `json:"tenantId" bson:"tenantId"`
	LegalEntityID   string    `json:"legalEntityId" bson:"legalEntityId"`
	Code            string    `json:"code" bson:"code"`
	Name            string    `json:"name" bson:"name"`
	Description     string    `json:"description" bson:"description"`
	Status          string    `json:"status" bson:"status"`
	DataScope       string    `json:"dataScope" bson:"dataScope"`
	PermissionCodes []string  `json:"permissionCodes" bson:"permissionCodes"`
	UserCount       int       `json:"userCount" bson:"userCount,omitempty"`
	BuiltIn         bool      `json:"builtIn" bson:"builtIn"`
	CreatedAt       time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" bson:"updatedAt"`
}

// Permission 可递归嵌套，ParentID 仅用于持久化，不下发到前端。
//
// Scope 声明权限生效层级：platform（平台运营）、tenant（租户）、entity（经营主体）。
type Permission struct {
	ID        string       `json:"id" bson:"_id"`
	Name      string       `json:"name" bson:"name"`
	Code      string       `json:"code" bson:"code"`
	Type      string       `json:"type" bson:"type"`
	Scope     string       `json:"scope" bson:"scope"`
	ParentID  string       `json:"-" bson:"parentId"`
	SortOrder int          `json:"-" bson:"sortOrder"`
	Children  []Permission `json:"children,omitempty" bson:"-"`
}

// AuthSession 是登录/恢复会话的响应体，同时下发可切换的租户与经营主体。
type AuthSession struct {
	Account              Account       `json:"account"`
	Permissions          []string      `json:"permissions"`
	Tenants              []Tenant      `json:"tenants"`
	CurrentTenantID      string        `json:"currentTenantId"`
	LegalEntities        []LegalEntity `json:"legalEntities"`
	CurrentLegalEntityID string        `json:"currentLegalEntityId"`
	DataScope            string        `json:"dataScope"`
}

// RefreshSession 是刷新令牌在 Redis 中关联的会话信息。
type RefreshSession struct {
	AccountID     string `json:"accountId"`
	TenantID      string `json:"tenantId"`
	LegalEntityID string `json:"legalEntityId"`
}

// TokenPair 是登录与刷新后签发的令牌组合。
type TokenPair struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// Principal 是令牌校验后从仓储实时加载的调用主体。
type Principal struct {
	AccountID      string
	TenantID       string
	LegalEntityID  string
	DataScope      string
	LegalEntityIDs []string
	Permissions    []string
	RoleNames      []string
	PlatformAdmin  bool
}

// 状态与枚举常量集中定义，避免各处散落魔法字符串。
const (
	// TenantModeGroup 表示集团统一采购，租户下可挂多个经营主体。
	TenantModeGroup = "group"
	// TenantModeStandalone 表示单体公司采购，租户下只有一个经营主体。
	TenantModeStandalone = "standalone"

	// LegalEntityTypeGroup 表示集团总部主体，可作为其它主体的父级。
	LegalEntityTypeGroup = "GROUP"
	// LegalEntityTypeCompany 表示独立经营的法人公司。
	LegalEntityTypeCompany = "COMPANY"

	// DataScopeTenant 可查看租户下全部经营主体。
	DataScopeTenant = "tenant"
	// DataScopeEntity 仅可查看被授权的经营主体。
	DataScopeEntity = "entity"
	// DataScopeSelf 仅可查看本人组织及下级。
	DataScopeSelf = "self"

	// PermissionScopePlatform 为平台运营权限。
	PermissionScopePlatform = "platform"
	// PermissionScopeTenant 为租户级权限。
	PermissionScopeTenant = "tenant"
	// PermissionScopeEntity 为经营主体级权限。
	PermissionScopeEntity = "entity"

	// UserActive 表示账户启用。
	UserActive = "active"
	// UserDisabled 表示账户停用。
	UserDisabled = "disabled"
	// TenantActive 表示租户或主体正常。
	TenantActive = "active"
	// TenantSuspended 表示租户或主体暂停。
	TenantSuspended = "suspended"
)
