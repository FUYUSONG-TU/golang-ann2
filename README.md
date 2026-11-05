# 机匠学徒 · 后台管理系统

微信云托管 Golang 模板的二次开发版本。前端是完整重写的 Vue 3 单页应用，后端在保留模板数据层的基础上补齐了账号、角色、权限、日志、配置等后台基础设施；小程序端尚未开发，接口契约与页面骨架已预留。

| 层 | 选型 |
| --- | --- |
| 前端 | Vue 3 + Vite 5 + Element Plus + vue-router（history 模式） |
| 后端 | Go 标准库 `net/http` + GORM（无鉴权第三方依赖，口令与令牌均为标准库实现） |
| 数据库 | MySQL，首次启动自动建表并写入内置数据 |
| 部署 | 前端产物经 `go:embed` 嵌入 Go 二进制，单容器运行 |

---

## 一、功能清单

### 已实现

- **完整登录流程**：账号密码登录、记住我（7 天）、退出登录、修改密码、登录失败留痕、禁用账号拦截、令牌过期自动跳登录页并回跳原地址。
- **三层权限模型**：账号 → 角色 → 权限编码。菜单、路由、接口三处都会校验，越权访问会写进操作日志。
- **账号管理**：分页、按账号/昵称/手机号搜索、按角色与状态筛选、新建、编辑、重置密码、启用禁用、删除；不能删除自己、不能禁用自己、至少保留一个超级管理员。
- **角色权限**：内置三个角色（超级管理员 / 运营人员 / 只读访客）+ 自定义角色；按分组勾选 14 项权限；内置角色不可删除、角色下有账号时不可删除。
- **操作日志**：登录日志与操作日志分标签页；写操作自动记录（账号、模块、动作、方法、路径、IP、结果、失败原因）；可按关键词、模块、结果筛选，可整体清空。
- **数据看板**：账号/登录/操作/计数四类指标、近 7 天登录趋势、最近登录与最近操作、数据库连通与延迟、运行时长。
- **系统设置**：站点名称、标语、Logo 文字、版权、备案号、后台公告、维护模式、小程序名称与 AppID，按分组落库。
- **系统信息**：Go 版本、架构、协程数、运行时长、MySQL 版本、连接延迟、部署提示。
- **个人中心**：账号信息、权限清单、修改密码（改后强制重新登录）。
- **小程序骨架**：8 个功能模块与接口路径的清单、就绪状态、接入路线步骤条。

### 待办（下一步）

- 小程序端开发与 `/api/mp/*` 真实实现（当前只有契约与页面骨架）。
- 头像上传、文件存储。
- 令牌刷新与登出黑名单（当前为无状态令牌，改密后旧令牌仍会自然过期）。
- 小程序用户与后台账号的绑定关系。

---

## 二、目录结构

```
.
├── main.go                  # 入口：初始化密钥、数据库、路由
├── static.go                # 前端资源嵌入 + SPA history 回退
├── service/                 # 接口层
│   ├── admin_service.go     # 路由表与分发、权限校验、操作日志
│   ├── auth.go              # 登录/登出/资料/改密、令牌解析、上下文
│   ├── users.go             # 账号 CRUD
│   ├── roles.go             # 角色与权限字典
│   ├── logs.go              # 日志查询与清空
│   ├── configs.go           # 站点配置
│   ├── dashboard.go         # 看板、系统信息、小程序骨架
│   ├── counters.go          # 计数业务（模板示例数据）
│   ├── response.go          # 统一响应与参数工具
│   └── counter_service.go   # 模板自带的 /api/count
├── pkg/security/            # PBKDF2-SHA256 口令散列 + HMAC 访问令牌
├── db/
│   ├── init.go              # 连接池、AutoMigrate、内置数据种子
│   ├── model/               # Counter / AdminUser / AdminRole / 日志 / 配置
│   └── dao/                 # 数据访问
├── web/                     # Vue 前端工程
│   ├── src/views/           # 11 个页面
│   ├── src/layouts/         # 侧边栏 + 顶栏
│   ├── src/menu.js          # 菜单与权限模块的唯一定义
│   └── dist/                # 构建产物，不提交
└── Dockerfile               # 三段构建：node → go → alpine
```

---

## 三、页面与菜单

| 路径 | 页面 | 需要的权限 |
| --- | --- | --- |
| `/login` | 登录 | 无需登录 |
| `/` | 数据看板 | `dashboard` |
| `/users` | 账号管理 | `users.view` / `users` |
| `/roles` | 角色权限 | `roles.view` / `roles` |
| `/counters` | 计数管理 | `counters.view` / `counters` |
| `/mini` | 小程序管理（骨架） | `mp.view` / `mp` |
| `/logs` | 操作日志 | `logs.view` / `logs` |
| `/settings` | 系统设置 | `configs.view` / `configs` |
| `/system` | 系统信息 | `system:view` |
| `/profile` | 个人中心 | 登录即可 |
| `/403` | 无权限提示 | 登录即可 |

菜单由 `web/src/menu.js` 统一声明，按当前账号权限自动收敛；路由守卫与后端接口各自独立校验，前端隐藏菜单只是体验，真正的拦截在后端。

---

## 四、权限编码

| 编码 | 名称 | 说明 |
| --- | --- | --- |
| `dashboard` | 数据看板 | 查看看板与统计 |
| `users.view` | 账号查看 | 只看列表 |
| `users` | 账号管理 | 新建/编辑/禁用/删除/重置密码 |
| `roles.view` | 角色查看 | 只看角色 |
| `roles` | 角色管理 | 新建/编辑/删除角色 |
| `counters.view` | 计数查看 | 只看计数 |
| `counters` | 计数管理 | 增删改计数 |
| `mp.view` | 小程序查看 | 看骨架 |
| `mp` | 小程序管理 | 维护小程序内容 |
| `logs.view` | 日志查看 | 查登录/操作日志 |
| `logs` | 日志管理 | 清空日志 |
| `configs.view` | 设置查看 | 只看设置 |
| `configs` | 设置管理 | 修改设置 |
| `system:view` | 系统信息 | 运行时与数据库状态 |

约定：`模块` = 该模块的写权限，`模块.view` = 只读权限；`*` 表示全部（超级管理员固定拥有）。

内置角色：

| 角色 | 编码 | 权限 |
| --- | --- | --- |
| 超级管理员 | `super` | `*` |
| 运营人员 | `operator` | `dashboard, users.view, counters, mp, logs.view` |
| 只读访客 | `viewer` | `dashboard, counters.view, logs.view` |

---

## 五、登录与安全

- 口令使用 **PBKDF2-HMAC-SHA256**（12000 次迭代、16 字节随机盐）存储，格式 `pbkdf2_sha256$迭代$盐$散列`，不引入第三方依赖。
- 登录成功后签发 **HMAC-SHA256 访问令牌**（`base64url(claims).base64url(sig)`），默认 12 小时，勾选「记住我」为 7 天。
- 令牌签名密钥取 `ADMIN_SECRET`；未配置时回退到 `ADMIN_TOKEN`，再回退到由 `MYSQL_PASSWORD` 派生的值，因此多副本部署无需额外配置即可互通。**公网环境建议显式配置 `ADMIN_SECRET`。**
- 首次启动自动创建管理员 `admin`，初始密码 `admin123`（可用 `ADMIN_INIT_PASSWORD` 指定），并标记 `mustChangePwd`，看板与个人中心会持续提醒修改。
- 每次请求都会回查账号状态，禁用账号立即失效，无需等令牌过期。

### 环境变量

| 变量 | 必填 | 说明 |
| --- | --- | --- |
| `MYSQL_USERNAME` | 是 | 数据库账号 |
| `MYSQL_PASSWORD` | 是 | 数据库密码 |
| `MYSQL_ADDRESS` | 是 | `host:port`，容器内用内网地址 |
| `MYSQL_DATABASE` | 否 | 默认 `golang_demo` |
| `ADMIN_SECRET` | 建议 | 令牌签名密钥，多副本必须一致 |
| `ADMIN_INIT_PASSWORD` | 否 | 首次初始化管理员的密码 |
| `PORT` | 否 | 本地调试端口，容器内默认 80 |

---

## 六、数据表

启动时 `AutoMigrate` 自动创建，已存在则只补字段；内置角色、默认配置、初始管理员只在缺失时写入。

| 表 | 说明 |
| --- | --- |
| `admin_user` | 后台账号 |
| `admin_role` | 角色与权限串 |
| `admin_login_log` | 登录日志 |
| `admin_operation_log` | 操作日志 |
| `sys_config` | 站点配置（键值对） |
| `Counters` | 模板自带计数表 |

---

## 七、接口一览

统一前缀 `/api/admin`，响应结构 `{ code, errorMsg, data }`，`code` 为 0 表示成功。
除登录外均需请求头 `Authorization: Bearer <token>`。

### 登录态

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/auth/login` | `{username, password, remember}`，返回 token 与权限 |
| GET | `/auth/profile` | 当前账号、角色、权限、通用运行信息 |
| POST | `/auth/logout` | 记录退出日志 |
| POST | `/auth/password` | `{oldPassword, newPassword}` |

### 看板与系统

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/dashboard` | 指标、7 天趋势、最近登录与操作 |
| GET | `/system/info` | 运行时与数据库状态 |
| GET | `/mp/summary` | 小程序模块骨架 |

### 账号

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/users` | 分页查询，参数 `page/size/keyword/role/status`，同时返回角色选项 |
| POST | `/users/create` | `{username, password, nickname, roleCode, phone, email, remark}` |
| POST | `/users/update` | `{id, nickname, roleCode, phone, email, remark}` |
| POST | `/users/delete` | `{id}` |
| POST | `/users/status` | `{id, status}`，1 启用 / 0 禁用 |
| POST | `/users/reset-password` | `{id, password}` |

### 角色

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/roles` | 角色列表，含账号数 |
| GET | `/roles/permissions` | 权限字典（分组） |
| POST | `/roles/save` | `{id?, code, name, description, permissions[], sort}` |
| POST | `/roles/delete` | `{id}` |

### 日志 / 设置 / 计数

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/logs/login` | 参数 `page/size/keyword/success` |
| GET | `/logs/operation` | 参数 `page/size/keyword/module/success` |
| POST | `/logs/clear` | `{scope: login \| operation \| all}` |
| GET | `/configs` | 分组配置与扁平 values |
| POST | `/configs/save` | `{items: [{key, value}]}` |
| GET | `/counters` | 全部计数记录 |
| POST | `/counters/create` | `{count}`，可省略 |
| POST | `/counters/incr` | `{id, step}` |
| POST | `/counters/set` | `{id, count}` |
| POST | `/counters/delete` | `{id}` |

### 模板自带（无需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/count` | 读取当前计数 |
| POST | `/api/count` | `{"action":"inc"}` 自增 / `{"action":"clear"}` 清零 |
| GET | `/api/health` | 健康检查 |

---

## 八、本地开发

前端（热更新，`/api` 代理到后端 8080）：

```bash
cd web
npm install
npm run dev      # http://localhost:5173
```

后端（`web/dist` 必须存在，否则 `go:embed` 编译失败）：

```bash
cd web && npm run build && cd ..
export MYSQL_USERNAME=xxx MYSQL_PASSWORD=xxx
export MYSQL_ADDRESS=10.0.0.0:3306 MYSQL_DATABASE=golang_demo
export ADMIN_SECRET=your-secret
PORT=8080 go run .
```

本地不在云托管 VPC 内时，`MYSQL_ADDRESS` 要填 MySQL 的**外网地址**。

打包：

```bash
cd web && npm run build && cd ..
go build -ldflags="-s -w" -o main .
```

---

## 九、部署到微信云托管

1. 代码推到 GitHub 仓库，在云托管「部署发布 → 从代码仓库部署」选择分支。
2. 服务设置里补全 `MYSQL_*` 环境变量，并加上 `ADMIN_SECRET`。
3. 构建过程会自动执行 `npm install && npm run build` 与 `go build`，耗时比纯 Go 模板多 1~3 分钟。
4. 首次访问用 `admin / admin123` 登录，登录后立刻到「个人中心」改密码，或在部署前设置 `ADMIN_INIT_PASSWORD`。

> 服务设置里的「最小实例数」建议 ≥ 1，避免冷启动导致第一次登录等待过久。

---

## 十、常见问题

**登录一直提示 401**
检查 `ADMIN_SECRET` 是否在多次发布间被改动；改动后旧令牌会全部失效，重新登录即可。

**忘记管理员密码**
在数据库执行：
```sql
UPDATE admin_user SET password = '<新口令的散列>' WHERE username = 'admin';
```
更省事的做法是临时给服务加上 `ADMIN_INIT_PASSWORD` 后删除该表记录重启，让种子逻辑重建管理员。

**页面能打开但接口 403**
当前账号的角色缺少对应权限。用超级管理员进「角色权限」勾选即可，无需重新登录（权限在每次请求时实时读取）。

**新增页面后菜单不显示**
在 `web/src/menu.js` 加一项，并确认角色勾选了对应模块权限；路由还需要在 `web/src/router/index.js` 注册并写 `meta.module`。

---

## License

[MIT](./LICENSE)
