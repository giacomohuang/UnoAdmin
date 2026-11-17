package grpcapi

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"

	pb "github.com/unoadmin/admin-service/gen/adminbase/v1"
	"github.com/unoadmin/admin-service/internal/domain"
)

// toProtoUser 把领域账户转换为 gRPC 的 User 消息。
func toProtoUser(account domain.Account) *pb.User {
	return &pb.User{
		Id:          account.ID,
		Username:    account.Username,
		DisplayName: account.DisplayName,
		Email:       account.Email,
		Status:      account.Status,
		RoleNames:   account.RoleNames,
	}
}

// toProtoRole 把领域角色转换为 gRPC 消息。
func toProtoRole(role domain.Role) *pb.Role {
	return &pb.Role{
		Id:              role.ID,
		Name:            role.Name,
		Description:     role.Description,
		PermissionCodes: role.PermissionCodes,
		UserCount:       int32(role.UserCount),
	}
}

// toProtoPermission 递归转换权限节点为 gRPC 消息。
func toProtoPermission(node domain.Permission) *pb.Permission {
	result := &pb.Permission{
		Id:   node.ID,
		Name: node.Name,
		Code: node.Code,
		Type: node.Type,
	}
	for _, child := range node.Children {
		result.Children = append(result.Children, toProtoPermission(child))
	}
	return result
}

// toProtoTenant 把领域租户转换为 gRPC 消息。
func toProtoTenant(tenant domain.Tenant) *pb.Tenant {
	return &pb.Tenant{
		Id:     tenant.ID,
		Name:   tenant.Name,
		Code:   tenant.Code,
		Status: tenant.Status,
	}
}

// tokenFromContext 从 gRPC metadata 读取终端用户令牌。
// authorization 保留给服务间共享密钥，用户令牌使用独立的 x-user-token 头。
func tokenFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("x-user-token")
	if len(values) == 0 {
		return ""
	}
	return strings.TrimPrefix(values[0], "Bearer ")
}
