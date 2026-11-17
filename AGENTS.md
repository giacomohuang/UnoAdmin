# AGENTS.md

本文件给后续在本仓库工作的编码代理使用。变更前先读本文件，并优先遵循距离目标文件最近的 `AGENTS.md`。

## 项目结构

- 项目采用 monorepo 结构，工作区配置在 `pnpm-workspace.yaml`，当前包含 `packages/frontend/**`。
- `packages/backend/admin-service` 是 Go 服务，不在 pnpm 工作区内，单独用 `go build/test` 校验。
- 多租户领域模型与接口约定见 `docs/multi-tenant-model.md`，前端结构见 `docs/frontend-architecture.md`。

## 代码注释约定

- 新增或修改前端、后端、数据库及测试代码必须添加注释。
- 前端声明级注释统一使用 JSDoc，Go 声明级注释统一使用 Go doc 注释（`// Name ...`）。

## 领域术语

- **Tenant（租户）**：采购、计费与数据隔离的最外层，`mode` 区分集团与单体。
- **LegalEntity（经营主体）**：租户内相互隔离的法人实体，集团总部为 `GROUP`，经营公司为 `COMPANY`。
- **Account（账户）**：管理端登录身份，不再使用 `User` 命名。
- **数据边界由后端强制**：`tenantId` / `legalEntityId` 只能来自令牌，前端不得自行推导或透传不可信范围。

## 后端约定

- gRPC 契约在 `api/proto/adminbase/v1/adminbase.proto`，对外命名同样使用 Account，不引入 User；修改 proto 后必须重新生成 `gen/adminbase/v1`：

  ```sh
  cd packages/backend/admin-service
  protoc --go_out=gen --go_opt=module=github.com/unoadmin/admin-service/gen \
    --go-grpc_out=gen --go-grpc_opt=module=github.com/unoadmin/admin-service/gen \
    -Iapi/proto api/proto/adminbase/v1/adminbase.proto
  ```

## 前端约定

- UI 组件统一使用 `@mcistudio/unoui-vue`，禁止引入其他组件库或自建基础控件。
- 组件按子路径导入：`@mcistudio/unoui-vue/table`、`@mcistudio/unoui-vue/select`。
- 样式使用 UnoCSS 原子类与语义色（`bg-primary`、`text-secondary`、`border-light`、`brand`），不新增全局 CSS 类。
- 图标使用 Iconify 类名（`i-lucide:plus`），不引入图标字体或 SVG 组件库。
- 页面路由由 `routes-generator` 生成：文件路径决定 URL，`<router lang="json">` 声明 `title` 与 `permission`。
- 菜单、路由、按钮必须使用同一批权限码；新增权限时同步更新 `admin-base/src/index.ts`、页面 meta 与后端 seed。

## 前端 packages 安装和维护

- 不要手动修改 `package.json` 和 `pnpm-lock.yaml`，而是使用 `pnpm add <package>`、`pnpm install` 安装。
- 本机 `pnpm add` 若因符号链接失败，可先 `pnpm add --lockfile-only` 更新清单，再 `pnpm install --force` 重建链接。
