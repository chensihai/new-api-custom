# Release Notes v1.4.0

## 🎯 重大变更：经典前端操练场 → 游乐场模块替换

将经典前端（web/classic）中的"操练场"模块整体替换为新版前端（web/default）中的"游乐场"模块，带来全新的 UI 体验和更丰富的交互功能。

---

## ✨ 新功能

### 游乐场（Playground）全新体验
- **现代化 UI 组件**：采用 shadcn/ui + ai-elements 组件库，替代原 Semi Design 自建组件，界面更简洁美观
- **模型/分组选择器**：全新的 `ModelGroupSelector` 组件，支持搜索过滤、桌面端 Popover + 移动端 Drawer 双端适配
- **AI 对话组件**：基于 ai-elements 的 Conversation、Message、Reasoning 等组件，原生支持推理内容展示、消息分支、消息操作
- **流式对话优化**：使用 `sse.js` + 自定义 `useStreamRequest` Hook，SSE 流式渲染更流畅
- **TanStack Query 状态管理**：模型列表和分组列表使用 TanStack Query 缓存，减少重复请求，提升加载速度
- **参数配置面板**：Temperature、TopP、MaxTokens 等参数的开关式配置，支持持久化到 localStorage
- **sonner Toast 通知**：替代原 react-toastify，与新版游乐场 UI 风格统一

### 导航与 i18n
- 侧边栏菜单项从"操练场"更名为"游乐场"，图标更新为 FlaskConical
- 8 个语言文件（zh/en/fr/ja/ru/vi/zh-CN/zh-TW）同步更新翻译

---

## 🔒 安全修复

| 编号 | 严重程度 | 问题描述 | 修复方式 |
|------|---------|---------|---------|
| SEC-1 | 🔴 严重 | localStorage 存储完整用户对象（含 OAuth IDs、Stripe ID 等敏感信息） | 写入时仅保留最小字段（id/username/display_name/role/status/group） |
| SEC-2 | 🟠 高 | `getCommonHeaders()` 中 uid 直接从 localStorage 读取无校验，可被篡改 | 添加 `/^\d+$/` 正则校验，仅接受纯数字 uid |
| ADAPT-1 | 🟠 高 | `api-adapter` 使用 `New-Api-User`，`helpers/api.js` 使用 `New-API-User`，Header 大小写不一致 | 统一为 `New-API-User` |

---

## 🐛 Bug 修复

| 编号 | 问题描述 | 修复方式 |
|------|---------|---------|
| BUG-1 | QueryClient 模块级单例在 React StrictMode 下可能缓存异常 | 改用 `useRef` + 惰性单例模式 |
| BUG-2 | `loadUserFromStorage` 无类型校验，`JSON.parse` 结果可能不符合接口 | 添加 `isSafeUser()` 运行时类型校验 |
| BUG-3 | `PageLayout.loadUser` 和 `getUserIdFromLocalStorage` 中 `JSON.parse` 无 try-catch | 添加 try-catch 保护，损坏数据自动清理 |
| ADAPT-2 | Playground 页面内和 PageLayout 各有 `<Toaster />`，Toast 重复显示 | 移除 Playground 页面内重复的 Toaster |

---

## 📦 依赖变更

### 新增依赖
- `@tanstack/react-query@^5.95.2` — 数据请求缓存
- `sonner@^2.0.7` — Toast 通知
- `nanoid@^5.1.6` — 唯一 ID 生成
- `@base-ui/react@^1.4.1` — UI 组件库基础
- `cmdk@^1.1.1` — Command 搜索组件
- `vaul@^1.1.2` — Drawer 抽屉组件
- `motion@^12.38.0` — 动画库
- `tailwind-merge@^3.5.0` — Tailwind 类名合并
- `class-variance-authority@^0.7.1` — 组件变体管理
- `tailwindcss-animate@^1.0.7` — Tailwind 动画插件
- `use-stick-to-bottom@^1.1.1` — 自动滚动 Hook
- `streamdown@^2.0.1` — 流式 Markdown 渲染
- `ai@^6.0.27` — AI UI 组件辅助

### 升级依赖
- `sse.js`: `^2.6.0` → `^2.7.2`
- `typescript`: `4.4.2` → `~5.5.0`

---

## 🏗️ 构建与配置

- **tsconfig.json**：新增 TypeScript 配置（allowJs、jsx: react-jsx、路径别名）
- **vite.config.js**：
  - `resolve.alias` 改为数组形式，添加 `@/lib/api` 和 `@/stores/auth-store` 别名劫持到适配层
  - `optimizeDeps.include` 添加新依赖预构建
  - `manualChunks` 添加 `playground` 专用分包
- **tailwind.config.js**：添加 12 个 shadcn/ui CSS 变量映射到 Semi Design 变量，新增 `tailwindcss-animate` 插件

---

## 🗑️ 移除内容

- `components/playground/` — 18 个旧操练场子组件
- `hooks/playground/` — 6 个旧操练场 Hooks
- `contexts/PlaygroundContext.jsx` — 旧 Context
- `constants/playground.constants.js` — 旧常量
- `helpers/api.js` 中 `buildApiPayload`、`processModelsData`、`processGroupsData`、`handleApiError`
- `helpers/utils.jsx` 中 `getTextContent`、`processThinkTags`、`formatMessageForAPI` 等 12 个操练场专用函数

> 注：`CodeViewer.jsx` 保留至 `components/` 目录，因 `NotificationSettings` 仍依赖此组件。

---

## 🔄 数据兼容性

- **localStorage 键名不变**：`playground_config`、`playground_messages`、`playground_parameter_enabled` 与旧版一致
- **API 端点不变**：`/pg/chat/completions`、`/api/user/models`、`/api/user/self/groups`
- **路由不变**：`/console/playground`（PrivateRoute 鉴权不变）
- **权限控制不变**：`chat.playground` 模块可见性开关仍生效

用户历史数据（配置、消息记录）可无缝迁移至新版游乐场。
