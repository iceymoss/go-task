# 数据库模型文档

## 概述

本项目使用 MySQL 和 MongoDB 双数据库架构：
- **MySQL**: 存储结构化数据，包括用户、任务、执行记录、告警等核心业务数据
- **MySQL**: 22 张核心表
- **MongoDB**: 存储日志、统计、报告等海量数据和分析数据
- **MongoDB**: 5 个核心集合

---

## MySQL 数据表 (22张)

### 1. 用户和权限 (4张表)

#### 1.1 users - 用户表
- **用途**: 存储用户账户信息
- **关键字段**: username, email, role, password_hash, status
- **关系**: 一对多到 sessions, audit_logs

#### 1.2 sessions - 会话表
- **用途**: 管理用户登录会话
- **关键字段**: user_id, token, expires_at
- **关系**: 多对一到 users

#### 1.3 roles - 角色表
- **用途**: 定义系统角色和权限
- **关键字段**: name, permissions
- **角色类型**: admin, operator, viewer

#### 1.4 user_roles - 用户角色关联表
- **用途**: 用户与角色多对多关联
- **关键字段**: user_id, role_id

---

### 2. 任务管理 (4张表)

#### 2.1 sys_jobs - 任务表
- **用途**: 存储任务定义和配置
- **关键字段**: name, type, cron_expr, enable, params
- **任务类型**: shell, http, email, sql, custom

#### 2.2 sys_job_groups - 任务分组表
- **用途**: 任务分组和层级管理
- **关键字段**: name, parent_id, path, level
- **特点**: 支持树形结构

#### 2.3 sys_job_versions - 任务版本表
- **用途**: 记录任务配置变更历史
- **关键字段**: job_id, version, config, change_log

#### 2.4 sys_param_templates - 参数模板表
- **用途**: 定义任务参数模板和Schema
- **关键字段**: name, task_type, params_schema, default_params

---

### 3. 执行记录 (2张表)

#### 3.1 sys_job_executions - 执行记录表
- **用途**: 记录每次任务执行情况
- **关键字段**: execution_id, job_id, status, duration_ms, retry_count
- **状态**: pending, running, success, failed, timeout, cancelled

#### 3.2 sys_job_logs - 执行日志表
- **用途**: 存储任务执行的详细日志
- **关键字段**: execution_id, log_level, message, timestamp
- **日志级别**: debug, info, warning, error

---

### 4. 告警管理 (4张表)

#### 4.1 sys_alert_rules - 告警规则表
- **用途**: 定义告警触发条件
- **关键字段**: job_id, alert_type, condition, threshold_count
- **告警类型**: failure, timeout, retry_exceeded, success, duration_exceeded

#### 4.2 sys_alert_channels - 告警渠道表
- **用途**: 配置告警通知渠道
- **关键字段**: name, channel_type, config
- **渠道类型**: email, sms, dingtalk, wechat, feishu, slack, webhook

#### 4.3 sys_alert_history - 告警历史表
- **用途**: 记录所有告警事件
- **关键字段**: alert_id, rule_id, alert_type, status, triggered_at

#### 4.4 sys_alert_silences - 告警静默表
- **用途**: 配置告警静默规则
- **关键字段**: job_id, alert_type, start_time, end_time

---

### 5. 工作流 (3张表)

#### 5.1 sys_workflows - 工作流定义表
- **用途**: 存储工作流DAG定义
- **关键字段**: workflow_id, dag, global_params, failure_strategy
- **失败策略**: fail_fast, continue, retry_all

#### 5.2 sys_workflow_executions - 工作流执行表
- **用途**: 记录工作流执行实例
- **关键字段**: execution_id, workflow_id, status, node_status

#### 5.3 sys_workflow_node_executions - 节点执行表
- **用途**: 记录工作流节点执行详情
- **关键字段**: workflow_exec_id, node_id, job_id, status

---

### 6. 模板系统 (3张表)

#### 6.1 sys_task_templates - 任务模板表
- **用途**: 预定义任务模板
- **分类**: data_sync, data_clean, backup, monitoring

#### 6.2 sys_workflow_templates - 工作流模板表
- **用途**: 预定义工作流模板
- **分类**: etl, data_pipeline, ml_pipeline

#### 6.3 sys_composite_templates - 复合模板表
- **用途**: 组合多个模板为复合模板
- **特点**: 支持任务和工作流的组合

---

### 7. 系统管理 (2张表)

#### 7.1 sys_audit_logs - 审计日志表
- **用途**: 记录所有用户操作
- **关键字段**: user_id, action, resource, old_value, new_value

#### 7.2 sys_configs - 系统配置表
- **用途**: 存储系统配置项
- **关键字段**: key, value, type, group, sensitive

#### 7.3 sys_notifications - 通知表
- **用途**: 系统通知管理
- **关键字段**: receiver_id, type, status, priority

---

## MongoDB 集合 (5个)

### 1. log_aggregations - 日志聚合集合
- **用途**: 聚合统计日志数据
- **特点**: 按时间窗口聚合，支持分钟/小时/天粒度
- **索引**: job_id, execution_id, window_start

### 2. realtime_stats - 实时统计集合
- **用途**: 存储实时统计数据
- **统计类型**: job, workflow, worker, system
- **特点**: 支持时间序列数据存储

### 3. execution_log_streams - 执行日志流集合
- **用途**: 存储海量执行日志
- **特点**: 支持流式写入和实时查询
- **TTL**: 可配置自动过期

### 4. report_data - 报告数据集合
- **用途**: 存储生成的报告数据
- **报告类型**: daily, weekly, monthly, custom
- **包含**: 摘要统计、图表配置、详细指标

### 5. event_timelines - 事件时间线集合
- **用途**: 记录系统事件时间线
- **事件类型**: job_created, job_updated, execution_started, alert_triggered
- **特点**: 支持事件追溯和时间线展示

---

## 数据关系图

```
users (1) ──< sessions (N)
      │
      └──< (N) user_roles (N) >── (1) roles

job_groups (1) ──< jobs (N)
       │
       └──> (tree structure)

jobs (1) ──< job_versions (N)
  │
  ├──> sys_param_templates (optional)
  │
  └──< sys_job_executions (N)
         │
         └──< sys_job_logs (N)

workflows (1) ──< workflow_executions (N)
                 │
                 └──< workflow_node_executions (N) >── (1) jobs (optional)

alerts
  ├── sys_alert_rules
  ├── sys_alert_channels
  ├── sys_alert_history
  └── sys_alert_silences

templates
  ├── sys_task_templates
  ├── sys_workflow_templates
  └── sys_composite_templates

system
  ├── sys_audit_logs
  ├── sys_configs
  └── sys_notifications
```

---

## 索引策略

### MySQL 索引
- **主键索引**: 所有表的自增ID
- **唯一索引**: 用户名、邮箱、任务名称、会话令牌、执行ID等
- **普通索引**: 外键关系、状态字段、时间字段
- **复合索引**: 频繁查询的组合字段

### MongoDB 索引
- **时间索引**: 所有集合的 timestamp 字段
- **业务索引**: job_id, execution_id, user_id 等
- **TTL索引**: execution_log_streams 自动过期
- **复合索引**: (job_id, timestamp), (user_id, created_at) 等

---

## 数据迁移

使用 `pkg/db/models/register.go` 中的 `RegisterMySQLModels()` 函数自动创建和更新所有表结构。

```go
import "your-project/pkg/db/models"

// 自动迁移所有MySQL表
err := models.RegisterMySQLModels(db)
```

---

## 性能优化建议

### MySQL
1. 执行记录表定期归档（保留最近3个月）
2. 日志表定期清理（保留最近1个月）
3. 审计日志按需保留（建议6个月）
4. 使用读写分离减轻主库压力

### MongoDB
1. 日志流集合设置TTL自动过期
2. 统计数据按时间分区
3. 报告数据定期归档
4. 使用合适的索引策略

---

## 数据一致性

1. **事务管理**: MySQL表使用事务保证一致性
2. **最终一致性**: MongoDB与MySQL之间采用异步同步
3. **数据校验**: 定期运行数据一致性检查任务
4. **备份策略**: MySQL每日全量备份 + 实时binlog，MongoDB每日快照

---

## 使用示例

```go
// 创建任务
job := &models.Job{
    Name:        "daily_backup",
    DisplayName:  "每日备份",
    Type:        "shell",
    CronExpr:    "0 2 * * *",
    Enable:      true,
}
db.Create(job)

// 查询执行记录
var executions []models.JobExecution
db.Where("job_id = ?", jobID).
   Order("created_at DESC").
   Limit(100).
   Find(&executions)

// MongoDB 写入日志
log := &mongomodels.ExecutionLogStream{
    ExecutionID: executionID,
    LogLevel:    "info",
    Message:     "任务执行完成",
    Timestamp:   time.Now(),
}
mongoDB.Collection("execution_log_streams").InsertOne(context.Background(), log)
```

---

## 更新日志

- 2026-03-12: 初始化所有模型，共22张MySQL表和5个MongoDB集合