// Package demo 实现示例业务包的后端用例，与前端 domain-demo 契约一一对应。
package demo

import (
	"context"

	"github.com/unoadmin/admin-service/internal/domain"
)

// Service 编排示例业务的查询逻辑。
type Service struct {
	repository domain.DemoRepository
}

// NewService 构造示例业务服务。
func NewService(repository domain.DemoRepository) *Service {
	return &Service{repository: repository}
}

// GetOverview 返回业务概览。
func (s *Service) GetOverview(ctx context.Context) (*domain.DemoOverview, error) {
	return s.repository.GetOverview(ctx)
}

// ListRecords 分页查询项目记录。
func (s *Service) ListRecords(ctx context.Context, query domain.ListQuery) (domain.PageResult[domain.DemoRecord], error) {
	return s.repository.ListRecords(ctx, query)
}
