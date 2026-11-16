package grpcapi_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/unoadmin/admin-service/gen/adminbase/v1"
	"github.com/unoadmin/admin-service/internal/domain"
	"github.com/unoadmin/admin-service/internal/grpcapi"
)

const serviceToken = "test-service-token"

// fakeVerifier 返回固定的调用主体。
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, raw string) (*domain.Principal, error) {
	if raw != "user-token" {
		return nil, domain.ErrUnauthorized
	}
	return &domain.Principal{
		UserID: "user-001", TenantID: "tenant-main", Permissions: []string{"user:read"},
	}, nil
}

// fakeAdmin 提供固定的查询结果。
type fakeAdmin struct{}

func (fakeAdmin) GetUser(context.Context, string) (domain.User, error) {
	return domain.User{ID: "user-001", Username: "lin.wei", RoleNames: []string{"平台管理员"}}, nil
}

func (fakeAdmin) ListUsers(_ context.Context, query domain.ListQuery) (domain.PageResult[domain.User], error) {
	return domain.PageResult[domain.User]{
		Items: []domain.User{{ID: "user-001", Username: "lin.wei", RoleNames: []string{"平台管理员"}}},
		Total: 1,
	}, nil
}

func (fakeAdmin) ListRoles(context.Context, domain.ListQuery) (domain.PageResult[domain.Role], error) {
	return domain.PageResult[domain.Role]{Items: []domain.Role{{ID: "role-admin", Name: "平台管理员"}}}, nil
}

func (fakeAdmin) ListPermissions(context.Context) ([]domain.Permission, error) {
	return []domain.Permission{{ID: "p-system", Code: "system", Children: []domain.Permission{{ID: "p-user", Code: "user:read"}}}}, nil
}

func (fakeAdmin) ListTenants(context.Context, domain.ListQuery) (domain.PageResult[domain.Tenant], error) {
	return domain.PageResult[domain.Tenant]{Items: []domain.Tenant{{ID: "tenant-main", Code: "HQ"}}}, nil
}

// startServer 在进程内启动 gRPC 服务并返回客户端连接。
func startServer(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcapi.ServiceTokenInterceptor(serviceToken)))
	grpcapi.NewServer(fakeVerifier{}, fakeAdmin{}).Register(server)

	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// serviceContext 附加服务间共享密钥。
func serviceContext() context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+serviceToken)
}

func TestServiceTokenInterceptorRejectsMissingToken(t *testing.T) {
	client := pb.NewAdminBaseClient(startServer(t))
	_, err := client.ListUsers(context.Background(), &pb.ListUsersRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("error = %v, want Unauthenticated", err)
	}
}

func TestVerifyTokenReturnsLivePermissions(t *testing.T) {
	client := pb.NewAdminBaseClient(startServer(t))
	resp, err := client.VerifyToken(serviceContext(), &pb.VerifyTokenRequest{Token: "user-token"})
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
	if !resp.GetValid() || resp.GetUserId() != "user-001" || resp.GetTenantId() != "tenant-main" {
		t.Fatalf("response = %+v, want valid user-001 / tenant-main", resp)
	}
	if len(resp.GetPermissions()) != 1 || resp.GetPermissions()[0] != "user:read" {
		t.Fatalf("permissions = %v, want user:read", resp.GetPermissions())
	}
}

func TestListUsersWithServiceToken(t *testing.T) {
	client := pb.NewAdminBaseClient(startServer(t))
	resp, err := client.ListUsers(serviceContext(), &pb.ListUsersRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if resp.GetTotal() != 1 || len(resp.GetItems()) != 1 || resp.GetItems()[0].GetRoleNames()[0] != "平台管理员" {
		t.Fatalf("response = %+v, want one user with role name", resp)
	}
}

func TestListPermissionsReturnsTree(t *testing.T) {
	client := pb.NewAdminBaseClient(startServer(t))
	resp, err := client.ListPermissions(serviceContext(), &pb.ListPermissionsRequest{})
	if err != nil {
		t.Fatalf("ListPermissions() error = %v", err)
	}
	if len(resp.GetItems()) != 1 || len(resp.GetItems()[0].GetChildren()) != 1 {
		t.Fatalf("response = %+v, want one root with one child", resp)
	}
}

func TestHasPermissionUsesUserToken(t *testing.T) {
	client := pb.NewAdminBaseClient(startServer(t))
	ctx := metadata.AppendToOutgoingContext(serviceContext(), "x-user-token", "Bearer user-token")
	resp, err := client.HasPermission(ctx, &pb.HasPermissionRequest{Permission: "user:read"})
	if err != nil {
		t.Fatalf("HasPermission() error = %v", err)
	}
	if !resp.GetAllowed() {
		t.Fatal("HasPermission() = false, want true")
	}
}
