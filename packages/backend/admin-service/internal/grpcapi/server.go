// Package grpcapi 暴露供其它微服务调用的 gRPC 接口。
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/unoadmin/admin-service/gen/adminbase/v1"
	"github.com/unoadmin/admin-service/internal/domain"
)

// verifier 是 gRPC 层需要的令牌校验能力。
type verifier interface {
	Verify(ctx context.Context, raw string) (*domain.Principal, error)
}

// adminAPI 是 gRPC 层需要的查询能力。
type adminAPI interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
	ListUsers(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.User], error)
	ListRoles(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Role], error)
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
	ListTenants(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.Tenant], error)
}

// Server 实现生成的 AdminBaseServer 接口。
type Server struct {
	pb.UnimplementedAdminBaseServer
	verifier verifier
	admin    adminAPI
}

// NewServer 构造 gRPC 服务实现。
func NewServer(v verifier, admin adminAPI) *Server {
	return &Server{verifier: v, admin: admin}
}

// Register 把当前实现注册到 gRPC Server。
func (s *Server) Register(server *grpc.Server) {
	pb.RegisterAdminBaseServer(server, s)
}

// VerifyToken 校验用户令牌并返回实时权限，供其它微服务鉴权。
func (s *Server) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	principal, err := s.verifier.Verify(ctx, req.GetToken())
	if err != nil {
		return &pb.VerifyTokenResponse{Valid: false, Error: err.Error()}, nil
	}
	return &pb.VerifyTokenResponse{
		Valid:       true,
		UserId:      principal.UserID,
		TenantId:    principal.TenantID,
		Permissions: principal.Permissions,
	}, nil
}

// GetUser 按 ID 返回用户。
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := s.admin.GetUser(ctx, req.GetId())
	if err != nil {
		return nil, grpcError(err)
	}
	return &pb.GetUserResponse{User: toProtoUser(user)}, nil
}

// ListUsers 分页返回用户，支持租户范围过滤。
func (s *Server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	page, err := s.admin.ListUsers(ctx, domain.ListQuery{
		Page: int(req.GetPage()), PageSize: int(req.GetPageSize()),
		Search: req.GetSearch(), TenantID: req.GetTenantId(),
	})
	if err != nil {
		return nil, grpcError(err)
	}
	items := make([]*pb.User, 0, len(page.Items))
	for _, user := range page.Items {
		items = append(items, toProtoUser(user))
	}
	return &pb.ListUsersResponse{Items: items, Total: int32(page.Total)}, nil
}

// ListRoles 分页返回角色。
func (s *Server) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	page, err := s.admin.ListRoles(ctx, domain.ListQuery{
		Page: int(req.GetPage()), PageSize: int(req.GetPageSize()), Search: req.GetSearch(),
	})
	if err != nil {
		return nil, grpcError(err)
	}
	items := make([]*pb.Role, 0, len(page.Items))
	for _, role := range page.Items {
		items = append(items, toProtoRole(role))
	}
	return &pb.ListRolesResponse{Items: items, Total: int32(page.Total)}, nil
}

// ListPermissions 返回权限树。
func (s *Server) ListPermissions(ctx context.Context, _ *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	nodes, err := s.admin.ListPermissions(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	items := make([]*pb.Permission, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, toProtoPermission(node))
	}
	return &pb.ListPermissionsResponse{Items: items}, nil
}

// ListTenants 分页返回租户。
func (s *Server) ListTenants(ctx context.Context, req *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error) {
	page, err := s.admin.ListTenants(ctx, domain.ListQuery{
		Page: int(req.GetPage()), PageSize: int(req.GetPageSize()), Search: req.GetSearch(),
	})
	if err != nil {
		return nil, grpcError(err)
	}
	items := make([]*pb.Tenant, 0, len(page.Items))
	for _, tenant := range page.Items {
		items = append(items, toProtoTenant(tenant))
	}
	return &pb.ListTenantsResponse{Items: items, Total: int32(page.Total)}, nil
}

// HasPermission 按调用方令牌中的实时权限做判定。
func (s *Server) HasPermission(ctx context.Context, req *pb.HasPermissionRequest) (*pb.HasPermissionResponse, error) {
	principal, err := s.verifier.Verify(ctx, tokenFromContext(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	allowed := false
	for _, permission := range principal.Permissions {
		if permission == "*" || permission == req.GetPermission() {
			allowed = true
			break
		}
	}
	return &pb.HasPermissionResponse{Allowed: allowed}, nil
}

// grpcError 将领域错误翻译为 gRPC 状态码。
func grpcError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
