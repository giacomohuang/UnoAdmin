// Package domain 定义与前端 core 对齐的业务实体、仓储端口与权限规则。
package domain

// ListQuery 对应前端 core 的 ListQuery，并额外支持租户范围过滤。
type ListQuery struct {
	Page     int
	PageSize int
	Search   string
	TenantID string
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

// User 与前端 core 的 User 接口字段一致，同时承载 Mongo 持久化字段。
// PasswordHash、RoleIDs、TenantIDs 只对后端可见，JSON 序列化时被忽略。
type User struct {
	ID           string   `json:"id" bson:"_id"`
	Username     string   `json:"username" bson:"username"`
	PasswordHash string   `json:"-" bson:"passwordHash"`
	DisplayName  string   `json:"displayName" bson:"displayName"`
	Email        string   `json:"email" bson:"email"`
	Status       string   `json:"status" bson:"status"`
	RoleIDs      []string `json:"-" bson:"roleIds"`
	TenantIDs    []string `json:"-" bson:"tenantIds"`
	RoleNames    []string `json:"roleNames" bson:"-"`
}

// Public 返回可安全下发到前端的用户副本，清除所有内部字段。
func (u User) Public() User {
	u.PasswordHash = ""
	u.RoleIDs = nil
	u.TenantIDs = nil
	if u.RoleNames == nil {
		u.RoleNames = []string{}
	}
	return u
}

// Role 与前端 core 的 Role 接口字段一致，UserCount 由仓储聚合得到。
type Role struct {
	ID              string   `json:"id" bson:"_id"`
	Name            string   `json:"name" bson:"name"`
	Description     string   `json:"description" bson:"description"`
	PermissionCodes []string `json:"permissionCodes" bson:"permissionCodes"`
	UserCount       int      `json:"userCount" bson:"userCount,omitempty"`
}

// Permission 可递归嵌套，ParentID 仅用于持久化，不下发到前端。
type Permission struct {
	ID        string       `json:"id" bson:"_id"`
	Name      string       `json:"name" bson:"name"`
	Code      string       `json:"code" bson:"code"`
	Type      string       `json:"type" bson:"type"`
	ParentID  string       `json:"-" bson:"parentId"`
	SortOrder int          `json:"-" bson:"sortOrder"`
	Children  []Permission `json:"children,omitempty" bson:"-"`
}

// Tenant 与前端 core 的 Tenant 接口字段一致。
type Tenant struct {
	ID     string `json:"id" bson:"_id"`
	Name   string `json:"name" bson:"name"`
	Code   string `json:"code" bson:"code"`
	Status string `json:"status" bson:"status"`
}

// AuthSession 对应前端 core 的 AuthSession，是登录/恢复会话的响应体。
type AuthSession struct {
	User            User     `json:"user"`
	Permissions     []string `json:"permissions"`
	Tenants         []Tenant `json:"tenants"`
	CurrentTenantID string   `json:"currentTenantId"`
}

// DemoOverview 对应 domain-demo 包的业务概览数据。
type DemoOverview struct {
	ActiveProjects int     `json:"activeProjects" bson:"activeProjects"`
	PendingTasks   int     `json:"pendingTasks" bson:"pendingTasks"`
	MonthlyGrowth  float64 `json:"monthlyGrowth" bson:"monthlyGrowth"`
}

// DemoRecord 对应 domain-demo 包的项目记录。
type DemoRecord struct {
	ID        string `json:"id" bson:"_id"`
	Name      string `json:"name" bson:"name"`
	Owner     string `json:"owner" bson:"owner"`
	Status    string `json:"status" bson:"status"`
	UpdatedAt string `json:"updatedAt" bson:"updatedAt"`
}

// RefreshSession 是刷新令牌在 Redis 中关联的会话信息。
type RefreshSession struct {
	UserID   string `json:"userId"`
	TenantID string `json:"tenantId"`
}

// TokenPair 是登录与刷新后签发的令牌组合。
type TokenPair struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// Principal 是令牌校验后从仓储实时加载的调用主体，包含当前有效权限。
type Principal struct {
	UserID      string
	TenantID    string
	Permissions []string
	RoleNames   []string
}

// Status 常量集中定义，避免各处散落魔法字符串。
const (
	UserActive      = "active"
	UserDisabled    = "disabled"
	TenantActive    = "active"
	TenantSuspended = "suspended"
)
