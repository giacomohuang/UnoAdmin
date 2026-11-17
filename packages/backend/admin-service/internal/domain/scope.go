package domain

import (
	"fmt"
	"sort"
)

// entityRank 用于在列表中把集团主体排在经营公司之前。
func entityRank(entity LegalEntity) int {
	if entity.Type == LegalEntityTypeGroup {
		return 0
	}
	return 1
}

// SortLegalEntities 按集团优先、创建时间稳定的顺序排列经营主体。
func SortLegalEntities(entities []LegalEntity) []LegalEntity {
	sorted := append([]LegalEntity(nil), entities...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if entityRank(sorted[i]) != entityRank(sorted[j]) {
			return entityRank(sorted[i]) < entityRank(sorted[j])
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

// ValidateTenantMode 校验租户形态与经营主体数量是否匹配。
//
// 单体租户必须且只能有一个经营主体，集团租户允许 0..n 个（创建时可以先建总部）。
func ValidateTenantMode(mode string, entityCount int) error {
	switch mode {
	case TenantModeStandalone:
		if entityCount > 1 {
			return fmt.Errorf("%w: standalone tenant can only own one legal entity", ErrInvalid)
		}
	case TenantModeGroup:
		// 集团租户对主体数量没有上限，保持向后兼容。
	default:
		return fmt.Errorf("%w: unsupported tenant mode %q", ErrInvalid, mode)
	}
	return nil
}

// ValidateDataScope 校验账户与角色的数据范围取值。
func ValidateDataScope(scope string) error {
	switch scope {
	case DataScopeTenant, DataScopeEntity, DataScopeSelf:
		return nil
	default:
		return fmt.Errorf("%w: unsupported data scope %q", ErrInvalid, scope)
	}
}

// WidenScope 在账户配置与角色授予中取更宽的数据范围。
//
// 顺序为 tenant > entity > self：角色不能收窄账户显式声明的主体范围，
// 但账户也不能凭自身配置突破角色的最小范围。
func WidenScope(left string, right string) string {
	rank := func(scope string) int {
		switch scope {
		case DataScopeTenant:
			return 2
		case DataScopeEntity:
			return 1
		default:
			return 0
		}
	}
	if rank(right) > rank(left) {
		return right
	}
	return left
}

// VisibleEntities 根据主体的租户与数据范围过滤经营主体。
//
// 平台管理员可查看租户下全部主体；租户级范围可查看全部主体；
// 主体级与自身级范围只能查看授权列表中的主体。
func VisibleEntities(principal *Principal, entities []LegalEntity) []LegalEntity {
	visible := make([]LegalEntity, 0, len(entities))
	for _, entity := range entities {
		if CanAccessEntity(principal, entity.ID) {
			visible = append(visible, entity)
		}
	}
	return visible
}

// CanAccessEntity 判断主体是否可访问指定经营主体。
func CanAccessEntity(principal *Principal, entityID string) bool {
	if principal == nil {
		return false
	}
	if entityID == "" {
		return true
	}
	if principal.PlatformAdmin || principal.DataScope == DataScopeTenant {
		return true
	}
	for _, id := range principal.LegalEntityIDs {
		if id == entityID {
			return true
		}
	}
	return false
}

// CanAccessTenant 判断主体是否可访问指定租户。
//
// 平台管理员不受租户限制；普通账户只能访问令牌中的租户。
func CanAccessTenant(principal *Principal, tenantID string) bool {
	if principal == nil {
		return false
	}
	if principal.PlatformAdmin {
		return true
	}
	return principal.TenantID == tenantID
}

// BuildOrgTree 将扁平组织节点组装为按 sort 排序的树。
// 指向不存在父节点的节点会被当作根节点，避免脏数据导致组织丢失。
func BuildOrgTree(nodes []OrgUnit) []OrgUnit {
	sorted := append([]OrgUnit(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sort == sorted[j].Sort {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].Sort < sorted[j].Sort
	})

	byID := make(map[string]OrgUnit, len(sorted))
	for _, node := range sorted {
		byID[node.ID] = node
	}
	childrenByParent := make(map[string][]OrgUnit)
	hasParent := make(map[string]bool)
	for _, node := range sorted {
		if node.ParentID == "" {
			continue
		}
		if _, ok := byID[node.ParentID]; !ok {
			continue
		}
		childrenByParent[node.ParentID] = append(childrenByParent[node.ParentID], node)
		hasParent[node.ID] = true
	}

	roots := make([]OrgUnit, 0)
	for _, node := range sorted {
		if hasParent[node.ID] {
			continue
		}
		roots = append(roots, withOrgChildren(node, childrenByParent))
	}
	return roots
}

// withOrgChildren 递归挂载子节点，形成完整的组织树。
func withOrgChildren(node OrgUnit, childrenByParent map[string][]OrgUnit) OrgUnit {
	children := childrenByParent[node.ID]
	if len(children) == 0 {
		return node
	}
	node.Children = make([]OrgUnit, 0, len(children))
	for _, child := range children {
		node.Children = append(node.Children, withOrgChildren(child, childrenByParent))
	}
	return node
}

// BuildEntityPath 根据父级主体计算新主体的路径，用于层级展示与循环校验。
func BuildEntityPath(parent *LegalEntity, selfID string) []string {
	if parent == nil {
		return []string{selfID}
	}
	path := append([]string(nil), parent.ID)
	return append(path, selfID)
}
