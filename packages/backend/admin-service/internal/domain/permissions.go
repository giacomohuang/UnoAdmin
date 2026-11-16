package domain

import "sort"

// EffectivePermissions 合并用户所有角色的权限码，去重并保持声明顺序。
// 任一角色持有通配符时收敛为 [ "*" ]，与前端 session.can 的语义保持一致。
func EffectivePermissions(roles []Role) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, role := range roles {
		for _, code := range role.PermissionCodes {
			if code == "*" {
				return []string{"*"}
			}
			if !seen[code] {
				seen[code] = true
				result = append(result, code)
			}
		}
	}
	return result
}

// Can 判断权限集合是否允许某个权限码。
// 空权限码视为无需授权，与前端 AdminSession.can(undefined) 的行为一致。
func Can(permissions []string, code string) bool {
	if code == "" {
		return true
	}
	for _, permission := range permissions {
		if permission == "*" || permission == code {
			return true
		}
	}
	return false
}

// BuildPermissionTree 将扁平权限节点按 sortOrder 组装为稳定有序的权限树。
// 指向不存在父节点的节点会被视为根节点，避免脏数据导致权限丢失。
func BuildPermissionTree(nodes []Permission) []Permission {
	sorted := append([]Permission(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder == sorted[j].SortOrder {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].SortOrder < sorted[j].SortOrder
	})

	byID := make(map[string]Permission, len(sorted))
	for _, node := range sorted {
		byID[node.ID] = node
	}
	childrenByParent := make(map[string][]Permission)
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

	roots := make([]Permission, 0)
	for _, node := range sorted {
		if hasParent[node.ID] {
			continue
		}
		roots = append(roots, withChildren(node, childrenByParent))
	}
	return roots
}

// withChildren 递归挂载子节点，形成完整的权限树。
func withChildren(node Permission, childrenByParent map[string][]Permission) Permission {
	children := childrenByParent[node.ID]
	if len(children) == 0 {
		return node
	}
	node.Children = make([]Permission, 0, len(children))
	for _, child := range children {
		node.Children = append(node.Children, withChildren(child, childrenByParent))
	}
	return node
}
