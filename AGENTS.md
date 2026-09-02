# AGENTS.md

本文件给后续在本仓库工作的编码代理使用。变更前先读本文件，并优先遵循距离目标文件最近的 `AGENTS.md`。

## 项目结构

- 项目采用monorepo结构，工作区配置在 `pnpm-workspace.yaml`，当前包含 `packages/frontend/**`。

## 代码注释约定

- 新增或修改前端、后端、数据库及测试代码必须添加注释。
- 前端声明级注释统一使用 JSDoc，Java 声明级注释统一使用 Javadoc。

## 前端packages安装和维护

- 不要手动修改packages.json和pnpm-lock.yaml，而是应该使用`pnpm add <package>`、`pnpm install` 安装。
