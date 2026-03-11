# Web UI 任务管理功能 - 实现总结

## 概述
本文档记录了为 go-task 项目添加的 Web UI 任务管理功能的完整实现。

## 已完成的功能

### Phase 1: 数据模型和后端 API ✅

#### 1. 数据模型 (pkg/db/models/job.go)
创建了完整的 Job 数据模型，包含：
- 基本信息：ID, Name, DisplayName, Type, CronExpr
- 配置信息：Params (JSON), Dependencies (JSON 数组)
- 优先级和重试：Priority, Timeout, MaxRetries
- 模板支持：IsTemplate, TemplateID
- 元数据：Description, Tags
- 时间戳：CreatedAt, UpdatedAt, LastRunAt, DeletedAt (软删除)

#### 2. 后端 API (internal/server/job_handler.go)
实现了完整的 RESTful API：

**任务管理 API:**
- `GET /api/jobs` - 获取任务列表（支持类型和状态筛选）
- `GET /api/jobs/:id` - 获取任务详情
- `POST /api/jobs` - 创建任务
- `PUT /api/jobs/:id` - 更新任务
- `DELETE /api/jobs/:id` - 删除任务（软删除）
- `POST /api/jobs/:id/enable` - 启用任务
- `POST /api/jobs/:id/disable` - 禁用任务

**任务操作 API:**
- `POST /api/jobs/:id/test` - 测试执行任务
- `GET /api/jobs/:id/logs` - 获取任务执行日志

**辅助功能 API:**
- `POST /api/jobs/validate-cron` - 验证 Cron 表达式并预测下次执行时间
- `GET /api/jobs/templates` - 获取任务模板列表
- `POST /api/jobs/from-template` - 从模板创建任务
- `GET /api/jobs/dependency-graph` - 获取任务依赖关系图

#### 3. 任务工厂 (internal/tasks/factory.go)
实现了通用的任务工厂，支持通过类型动态创建任务实例：
- shell - Shell 命令任务
- http - HTTP 请求任务
- email - 邮件发送任务
- sql - SQL 查询任务
- custom - 自定义任务（需注册）

#### 4. 通用任务实现

**Shell 任务** (internal/tasks/shell/shell.go)
- 支持执行 Shell 命令
- 支持工作目录设置
- 支持环境变量配置
- 支持命令超时
- 支持管道和逻辑运算符

**HTTP 任务** (internal/tasks/http/http.go)
- 支持 GET/POST/PUT/DELETE 等 HTTP 方法
- 支持自定义请求头
- 支持请求体（JSON）
- 支持超时配置
- 支持状态码验证

**Email 任务** (internal/tasks/email/email.go)
- 支持发送邮件
- 支持 CC/BCC
- 支持 HTML 和纯文本格式
- 支持自定义主题和正文

**SQL 任务** (internal/tasks/sql/sql.go)
- 支持 MySQL 数据库查询
- 支持执行 SQL 语句
- 返回影响行数
- 支持 SQL 参数配置

#### 5. 任务模板系统 (internal/tasks/template.go)
预设了 5 个常用任务模板：

1. **数据库备份** (backup_database)
   - Cron: 每天凌晨 2 点
   - 使用 mysqldump 备份数据库
   - 可配置：用户名、密码、数据库名、日期标识

2. **日志清理** (clean_logs)
   - Cron: 每周日凌晨 3 点
   - 清理过期日志文件
   - 可配置：日志目录、文件模式、保留天数

3. **健康检查** (health_check)
   - Cron: 每 5 分钟
   - HTTP 请求健康检查接口
   - 可配置：主机地址、端口

4. **发送日报** (send_report)
   - Cron: 工作日早上 8 点
   - 发送统计日报邮件
   - 可配置：收件人、日期、报告内容

5. **数据清理** (data_cleanup)
   - Cron: 每天凌晨 4 点
   - SQL 删除过期数据
   - 可配置：表名、保留天数

#### 6. 其他改进
- 添加了 TaskTypeWEB 常量 (pkg/constants/tasktype.go)
- 修复了 job_handler.go 中的 Cron 解析问题

## 数据库变更

需要执行数据库迁移以创建 sys_jobs 表：

```sql
CREATE TABLE `sys_jobs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `display_name` varchar(200) NOT NULL,
  `type` varchar(50) NOT NULL,
  `cron_expr` varchar(100) NOT NULL,
  `params` text,
  `dependencies` text,
  `priority` int DEFAULT '0',
  `timeout` int DEFAULT '3600',
  `max_retries` int DEFAULT '3',
  `is_template` tinyint(1) DEFAULT '0',
  `template_id` bigint unsigned DEFAULT NULL,
  `enable` tinyint(1) DEFAULT '1',
  `source` varchar(20) DEFAULT 'web',
  `description` text,
  `tags` text,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `last_run_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 前端页面功能

### 任务列表页面 (web/jobs.html)

**功能特性：**
- 任务列表展示（支持分页）
- 任务搜索和筛选（按类型、状态）
- 创建新任务
- 编辑现有任务
- 启用/禁用任务
- 测试执行任务
- 删除任务
- 从模板快速创建任务

**页面布局：**
- 工具栏：搜索框、类型筛选、状态筛选、创建按钮
- 任务表格：ID、名称、类型、Cron 表达式、状态、运行状态、上次运行时间、操作按钮
- 分页控制

### 任务详情页面 (web/job_detail.html)

**功能特性：**
- 任务基本信息展示
- 任务参数查看（JSON 格式）
- 下次运行时间计算
- 执行统计（总次数、成功、失败、平均耗时）
- 执行日志查看（最近执行和全部日志）
- 任务操作（启用/禁用、测试、编辑、删除）
- 自动刷新（30秒）

**页面布局：**
- 左侧列：任务信息卡片、任务参数卡片
- 右侧列：下次运行时间、统计卡片（4个）
- 底部：执行日志（带标签页切换）

## 快速开始

### 1. 启动服务

服务启动时会自动创建数据库表，无需手动执行 SQL。

```bash
cd /Users/iceymoss/project/go-task
go run cmd/scheduler/main.go
```

服务将启动在 `http://localhost:9099`（根据配置文件中的 `Server.Port`）。

### 2. 访问 Web 界面

- **登录页面**: http://localhost:9099/login.html
- **仪表盘**: http://localhost:9099/index.html
- **任务管理**: http://localhost:9099/jobs.html
- **任务详情**: http://localhost:9099/job_detail.html?id=1

### 3. 创建测试任务

**方法 1：通过 Web 界面**
1. 访问 http://localhost:9099/jobs.html
2. 点击"新建任务"或"从模板创建"
3. 填写任务信息并保存

**方法 2：通过 API**

创建一个简单的 Shell 任务：

```bash
curl -X POST http://localhost:9099/api/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "test_hello",
    "display_name": "测试任务",
    "type": "shell",
    "cron_expr": "* * * * *",
    "params": {
      "command": "echo Hello World"
    },
    "enable": true,
    "timeout": 60
  }'
```

### 4. 测试任务

**通过 Web 界面**：
1. 进入任务详情页
2. 点击"测试执行"按钮
3. 查看执行日志

**通过 API**：
```bash
curl -X POST http://localhost:9099/api/jobs/1/test \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 数据库表结构

服务启动时会自动创建以下表：

1. **sys_jobs** - 任务配置表
2. **sys_job_logs** - 任务执行日志表（已存在）
3. **users** - 用户表（已存在）
4. **sessions** - 会话表（已存在）

## 使用示例

### 1. 创建一个 Shell 任务

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily_backup",
    "display_name": "每日备份",
    "type": "shell",
    "cron_expr": "0 2 * * *",
    "params": {
      "command": "tar -czf /backup/app_$(date +%Y%m%d).tar.gz /app/data",
      "working_dir": "/"
    },
    "enable": true,
    "timeout": 3600
  }'
```

### 2. 创建一个 HTTP 健康检查任务

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "health_check",
    "display_name": "服务健康检查",
    "type": "http",
    "cron_expr": "*/5 * * * *",
    "params": {
      "url": "http://localhost:8080/health",
      "method": "GET",
      "expected_status": 200,
      "timeout": 30
    },
    "enable": true
  }'
```

### 3. 从模板创建任务

```bash
curl -X POST http://localhost:8080/api/jobs/from-template \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "backup_database",
    "name": "my_backup",
    "variables": {
      "username": "root",
      "password": "mypassword",
      "database": "myapp"
    },
    "enable": true
  }'
```

### 4. 获取任务列表

```bash
curl http://localhost:8080/api/jobs?type=shell&enable=true
```

### 5. 手动测试任务

```bash
curl -X POST http://localhost:8080/api/jobs/1/test
```

## 技术特性

### 1. 动态任务注册
- 支持通过 API 动态添加任务到调度器
- 任务创建后立即生效（如果 enable=true）
- 自动处理任务与调度器的同步

### 2. 任务依赖
- 支持配置任务间的依赖关系
- 依赖任务按顺序执行
- 提供 API 查询依赖关系图

### 3. 模板系统
- 预设常用任务模板
- 支持变量替换
- 可快速创建标准化任务

### 4. 重试机制
- 可配置最大重试次数
- 支持重试策略
- 失败日志记录

### 5. Cron 表达式验证
- 实时验证 Cron 表达式
- 预测下次执行时间
- 支持友好的错误提示

## 待实现功能

### Phase 2: 前端基础界面 ✅
- [x] 任务列表页面 (web/jobs.html)
- [x] 任务创建/编辑对话框
- [x] 任务详情页面 (web/job_detail.html)

### Phase 3: 高级功能
- [ ] Cron 表达式生成器（图形化界面）
- [ ] 任务依赖可视化（使用 D3.js 或类似库）
- [ ] 任务测试/调试模式（实时日志查看）

### Phase 4: 扩展功能
- [ ] 更多预设模板
- [ ] 自定义模板保存
- [ ] 任务分组/分类
- [ ] 任务执行统计和报表
- [ ] 邮件/钉钉/企业微信通知
- [ ] 任务导入/导出

## 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                   Web UI Frontend                     │
│  (待实现: Vue.js/React + Element UI/Ant Design)      │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP API
                      ▼
┌─────────────────────────────────────────────────────────┐
│              Job Handler (Gin Router)                  │
│  - CRUD Operations                                    │
│  - Task Execution                                    │
│  - Template Management                               │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│              Task Factory                              │
│  - Shell Task                                        │
│  - HTTP Task                                         │
│  - Email Task                                        │
│  - SQL Task                                          │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│              Scheduler Engine                          │
│  - Cron Scheduling                                  │
│  - Dependency Management                             │
│  - Retry Policy                                      │
│  - Event Handling                                   │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│              Database Layer                            │
│  - MySQL (GORM)                                     │
│  - Redis (可选)                                      │
│  - MongoDB (可选)                                    │
└─────────────────────────────────────────────────────────┘
```

## 配置说明

需要在配置文件中添加邮件服务器配置（用于 Email 任务）：

```yaml
# configs/config.yaml
email:
  host: "smtp.example.com"
  port: 587
  username: "your-email@example.com"
  password: "your-password"
```

## 注意事项

1. **安全性**
   - 任务参数中不要包含敏感信息（密码、密钥等）
   - 考虑使用环境变量或配置中心管理敏感配置
   - 对 API 接口进行认证授权

2. **性能**
   - Shell 任务可能阻塞调度器，建议设置合理的超时
   - 高频任务建议使用任务队列限流
   - 大量任务时考虑分页查询

3. **可观测性**
   - 利用现有的事件系统记录任务执行
   - 使用 StatManager 监控任务状态
   - 查看 sys_job_logs 表获取执行历史

## 总结

本次实现完成了 Web UI 任务管理的后端核心功能，包括：
- 完整的数据模型和 RESTful API
- 4 种通用任务类型（Shell、HTTP、Email、SQL）
- 任务模板系统，快速创建标准化任务
- 动态任务注册和调度器集成
- 任务依赖、重试、日志等功能

这些功能为后续的前端开发提供了坚实的基础，可以直接调用这些 API 构建用户友好的 Web 界面。