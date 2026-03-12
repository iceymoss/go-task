-- ============================================
-- Go-Task 数据库迁移脚本
-- 版本: 001
-- 描述: 创建所有核心表
-- ============================================

-- ============================================
-- 1. 用户和权限模块
-- ============================================

-- 1.1 角色权限表
CREATE TABLE IF NOT EXISTS `roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(50) NOT NULL COMMENT '角色名: admin, operator, viewer',
  `display_name` VARCHAR(100) NOT NULL COMMENT '显示名称',
  `description` TEXT COMMENT '描述',
  `permissions` JSON NOT NULL COMMENT '权限列表: ["job:create", "job:delete"]',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色权限表';

-- 1.2 用户角色关联表
CREATE TABLE IF NOT EXISTS `user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_role` (`user_id`, `role_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表';

-- ============================================
-- 2. 任务管理模块
-- ============================================

-- 2.1 任务分组表
CREATE TABLE IF NOT EXISTS `sys_job_groups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '分组标识',
  `display_name` VARCHAR(200) NOT NULL COMMENT '显示名称',
  `description` TEXT COMMENT '描述',
  `parent_id` BIGINT UNSIGNED COMMENT '父分组ID',
  `level` INT DEFAULT 1 COMMENT '层级',
  `path` VARCHAR(500) COMMENT '路径: /1/3/5',
  `sort` INT DEFAULT 0 COMMENT '排序',
  `icon` VARCHAR(50) COMMENT '图标',
  `color` VARCHAR(20) COMMENT '颜色',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_parent` (`parent_id`),
  KEY `idx_path` (`path`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务分组表';

-- 2.2 任务版本表
CREATE TABLE IF NOT EXISTS `sys_job_versions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id` BIGINT UNSIGNED NOT NULL COMMENT '任务ID',
  `version` VARCHAR(20) NOT NULL COMMENT '版本号',
  `config` JSON NOT NULL COMMENT '任务配置快照',
  `change_log` TEXT COMMENT '变更说明',
  `created_by` BIGINT UNSIGNED COMMENT '创建人',
  `created_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_job_version` (`job_id`, `version`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务版本表';

-- 2.3 原子任务模板表
CREATE TABLE IF NOT EXISTS `sys_task_templates` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `template_id` VARCHAR(64) NOT NULL COMMENT '模板ID',
  `name` VARCHAR(100) NOT NULL COMMENT '模板名称',
  `display_name` VARCHAR(200) NOT NULL COMMENT '显示名称',
  `description` TEXT COMMENT '描述',
  
  `task_type` VARCHAR(50) NOT NULL COMMENT '任务类型: shell, http, email, sql, ftp, docker, k8s',
  
  `params_schema` JSON NOT NULL COMMENT '参数Schema定义',
  `default_params` JSON COMMENT '默认参数值',
  
  `default_timeout` INT DEFAULT 3600 COMMENT '默认超时时间',
  `default_retry` INT DEFAULT 3 COMMENT '默认重试次数',
  
  `icon` VARCHAR(50) COMMENT '图标',
  `category` VARCHAR(50) COMMENT '分类',
  `tags` JSON COMMENT '标签',
  
  `is_public` TINYINT(1) DEFAULT 1 COMMENT '是否公开',
  `version` VARCHAR(20) DEFAULT '1.0.0',
  `created_by` BIGINT UNSIGNED COMMENT '创建人',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_template_id` (`template_id`),
  KEY `idx_task_type` (`task_type`),
  KEY `idx_category` (`category`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='原子任务模板表';

-- 2.4 更新 sys_jobs 表（添加新字段）
ALTER TABLE `sys_jobs` 
  ADD COLUMN IF NOT EXISTS `category` VARCHAR(50) DEFAULT 'default' COMMENT '任务分类: ops, data, ai, notification, workflow' AFTER `type`,
  ADD COLUMN IF NOT EXISTS `trigger_type` VARCHAR(20) DEFAULT 'cron' COMMENT '触发类型: cron, fixed_delay, fixed_rate, once, manual, webhook' AFTER `cron_expr`,
  ADD COLUMN IF NOT EXISTS `fixed_delay` INT COMMENT '固定延迟(秒)' AFTER `trigger_type`,
  ADD COLUMN IF NOT EXISTS `fixed_rate` INT COMMENT '固定频率(秒)' AFTER `fixed_delay`,
  ADD COLUMN IF NOT EXISTS `dependency_strategy` VARCHAR(20) DEFAULT 'strict' COMMENT '依赖策略: strict, ignore, wait' AFTER `dependencies`,
  ADD COLUMN IF NOT EXISTS `retry_backoff` INT DEFAULT 5 COMMENT '重试间隔(秒)' AFTER `max_retries`,
  ADD COLUMN IF NOT EXISTS `retry_strategy` VARCHAR(20) DEFAULT 'exponential' COMMENT '重试策略: fixed, exponential, random' AFTER `retry_backoff`,
  ADD COLUMN IF NOT EXISTS `concurrent_policy` VARCHAR(20) DEFAULT 'allow' COMMENT '并发策略: allow, skip, delay, deny' AFTER `retry_strategy`,
  ADD COLUMN IF NOT EXISTS `param_template_id` BIGINT UNSIGNED COMMENT '参数模板ID' AFTER `template_id`,
  ADD COLUMN IF NOT EXISTS `version` VARCHAR(20) DEFAULT '1.0.0' COMMENT '版本号' AFTER `description`,
  ADD COLUMN IF NOT EXISTS `created_by` BIGINT UNSIGNED COMMENT '创建人' AFTER `version`,
  ADD COLUMN IF NOT EXISTS `updated_by` BIGINT UNSIGNED COMMENT '更新人' AFTER `created_by`,
  ADD INDEX IF NOT EXISTS `idx_trigger_type` (`trigger_type`);

-- ============================================
-- 3. 执行记录模块
-- ============================================

-- 3.1 执行记录表（更新）
ALTER TABLE `sys_job_executions`
  ADD COLUMN IF NOT EXISTS `retry_reason` TEXT COMMENT '重试原因' AFTER `retry_count`,
  ADD COLUMN IF NOT EXISTS `trigger_type` VARCHAR(20) COMMENT '触发类型' AFTER `retry_reason`,
  ADD COLUMN IF NOT EXISTS `trigger_source` VARCHAR(50) COMMENT '触发来源: cron, manual, webhook, api' AFTER `trigger_type`,
  ADD COLUMN IF NOT EXISTS `output` TEXT COMMENT '标准输出' AFTER `error_stack`,
  ADD COLUMN IF NOT EXISTS `tags` JSON COMMENT '标签' AFTER `output`;

-- 3.2 执行日志表（更新）
ALTER TABLE `sys_job_logs`
  ADD COLUMN IF NOT EXISTS `job_id` BIGINT UNSIGNED COMMENT '任务ID' AFTER `execution_id`,
  ADD INDEX IF NOT EXISTS `idx_job_id` (`job_id`);

-- ============================================
-- 4. 监控告警模块
-- ============================================

-- 4.1 告警规则表
CREATE TABLE IF NOT EXISTS `sys_alert_rules` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '规则名称',
  `description` TEXT COMMENT '描述',
  `job_id` BIGINT UNSIGNED COMMENT '关联任务ID(为空表示全局规则)',
  `group_id` BIGINT UNSIGNED COMMENT '关联分组ID(为空表示全局)',
  
  `alert_type` VARCHAR(20) NOT NULL COMMENT '告警类型: failure, timeout, retry_exceeded, success, duration_exceeded',
  `condition` VARCHAR(20) NOT NULL COMMENT '触发条件: immediate, count_threshold, rate_threshold',
  `threshold_count` INT DEFAULT 1 COMMENT '阈值(次数)',
  `threshold_time` INT DEFAULT 300 COMMENT '时间窗口(秒)',
  `threshold_duration` INT COMMENT '时长阈值(毫秒)',
  
  `alert_level` VARCHAR(20) DEFAULT 'warning' COMMENT '告警级别: info, warning, error, critical',
  
  `silence_enabled` TINYINT(1) DEFAULT 0 COMMENT '是否启用静默',
  `silence_start` DATETIME(3) COMMENT '静默开始时间',
  `silence_end` DATETIME(3) COMMENT '静默结束时间',
  
  `enable` TINYINT(1) DEFAULT 1 COMMENT '是否启用',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  KEY `idx_job` (`job_id`),
  KEY `idx_group` (`group_id`),
  KEY `idx_alert_type` (`alert_type`),
  KEY `idx_enable` (`enable`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警规则表';

-- 4.2 告警通知渠道表
CREATE TABLE IF NOT EXISTS `sys_alert_channels` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '渠道名称',
  `channel_type` VARCHAR(20) NOT NULL COMMENT '渠道类型: email, sms, dingtalk, wechat, feishu, slack, webhook, telegram',
  `config` JSON NOT NULL COMMENT '渠道配置(加密)',
  `priority` INT DEFAULT 0 COMMENT '优先级',
  `enable` TINYINT(1) DEFAULT 1 COMMENT '是否启用',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  KEY `idx_type` (`channel_type`),
  KEY `idx_enable` (`enable`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警通知渠道表';

-- 4.3 告警历史表
CREATE TABLE IF NOT EXISTS `sys_alert_history` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `alert_id` VARCHAR(64) NOT NULL COMMENT '告警ID',
  `rule_id` BIGINT UNSIGNED COMMENT '规则ID',
  `job_id` BIGINT UNSIGNED COMMENT '任务ID',
  `execution_id` VARCHAR(64) COMMENT '执行ID',
  
  `alert_type` VARCHAR(20) NOT NULL COMMENT '告警类型',
  `alert_level` VARCHAR(20) DEFAULT 'warning' COMMENT '告警级别',
  `title` VARCHAR(200) COMMENT '告警标题',
  `message` TEXT COMMENT '告警消息',
  `details` JSON COMMENT '详细信息',
  
  `status` VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending, sending, sent, failed, cancelled',
  `channels` JSON COMMENT '发送的渠道',
  `failed_channels` JSON COMMENT '发送失败的渠道',
  
  `triggered_at` DATETIME(3) NOT NULL COMMENT '触发时间',
  `sent_at` DATETIME(3) COMMENT '发送时间',
  `created_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_alert_id` (`alert_id`),
  KEY `idx_rule` (`rule_id`),
  KEY `idx_job` (`job_id`),
  KEY `idx_execution` (`execution_id`),
  KEY `idx_status` (`status`),
  KEY `idx_triggered_at` (`triggered_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警历史表';

-- 4.4 告警静默表
CREATE TABLE IF NOT EXISTS `sys_alert_silences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '静默规则名称',
  `comment` TEXT COMMENT '说明',
  
  `job_id` BIGINT UNSIGNED COMMENT '任务ID',
  `job_name` VARCHAR(100) COMMENT '任务名称',
  `alert_type` VARCHAR(20) COMMENT '告警类型',
  `alert_level` VARCHAR(20) COMMENT '告警级别',
  `matchers` JSON COMMENT '匹配器: [{"type": "job", "value": "backup"}]',
  
  `start_time` DATETIME(3) NOT NULL COMMENT '开始时间',
  `end_time` DATETIME(3) NOT NULL COMMENT '结束时间',
  
  `status` VARCHAR(20) DEFAULT 'active' COMMENT '状态: active, expired, cancelled',
  `created_by` BIGINT UNSIGNED COMMENT '创建人',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  KEY `idx_job` (`job_id`),
  KEY `idx_status` (`status`),
  KEY `idx_time_range` (`start_time`, `end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警静默表';

-- ============================================
-- 5. 工作流模块
-- ============================================

-- 5.1 工作流定义表
CREATE TABLE IF NOT EXISTS `sys_workflows` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `workflow_id` VARCHAR(64) NOT NULL COMMENT '工作流ID',
  `name` VARCHAR(200) NOT NULL COMMENT '工作流名称',
  `description` TEXT COMMENT '描述',
  `version` VARCHAR(20) DEFAULT '1.0.0' COMMENT '版本号',
  
  `dag` JSON NOT NULL COMMENT 'DAG定义(节点和边)',
  `global_params` JSON COMMENT '全局参数',
  `schedule_config` JSON COMMENT '调度配置',
  
  `failure_strategy` VARCHAR(20) DEFAULT 'fail_fast' COMMENT '失败策略: fail_fast, continue, retry_all',
  
  `enable` TINYINT(1) DEFAULT 1 COMMENT '是否启用',
  `status` VARCHAR(20) DEFAULT 'active' COMMENT '状态: active, paused, archived',
  
  `tags` JSON COMMENT '标签',
  `author` VARCHAR(100) COMMENT '作者',
  `created_by` BIGINT UNSIGNED COMMENT '创建人',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_workflow_id` (`workflow_id`),
  KEY `idx_enable_status` (`enable`, `status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作流定义表';

-- 5.2 工作流执行记录表
CREATE TABLE IF NOT EXISTS `sys_workflow_executions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `execution_id` VARCHAR(64) NOT NULL COMMENT '执行ID',
  `workflow_id` VARCHAR(64) NOT NULL COMMENT '工作流ID',
  `workflow_name` VARCHAR(200) NOT NULL COMMENT '工作流名称',
  
  `status` VARCHAR(20) NOT NULL COMMENT '状态: pending, running, success, failed, cancelled, partial_success',
  
  `node_status` JSON NOT NULL COMMENT '节点执行状态',
  
  `scheduled_at` DATETIME(3) NOT NULL COMMENT '计划执行时间',
  `started_at` DATETIME(3) COMMENT '开始时间',
  `finished_at` DATETIME(3) COMMENT '完成时间',
  `duration_ms` BIGINT COMMENT '执行时长',
  
  `trigger_type` VARCHAR(20) COMMENT '触发类型',
  `trigger_source` VARCHAR(50) COMMENT '触发来源',
  
  `error_message` TEXT COMMENT '错误消息',
  `failed_nodes` JSON COMMENT '失败的节点列表',
  
  `metadata` JSON COMMENT '元数据',
  `tags` JSON COMMENT '标签',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_execution_id` (`execution_id`),
  KEY `idx_workflow` (`workflow_id`),
  KEY `idx_status` (`status`),
  KEY `idx_scheduled` (`scheduled_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作流执行记录表';

-- 5.3 工作流节点执行表
CREATE TABLE IF NOT EXISTS `sys_workflow_node_executions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `execution_id` VARCHAR(64) NOT NULL COMMENT '工作流执行ID',
  `node_id` VARCHAR(64) NOT NULL COMMENT '节点ID',
  `node_name` VARCHAR(200) NOT NULL COMMENT '节点名称',
  `node_type` VARCHAR(20) NOT NULL COMMENT '节点类型: job, parallel, condition, loop',
  `job_id` BIGINT UNSIGNED COMMENT '任务ID',
  `job_execution_id` VARCHAR(64) COMMENT '任务执行ID',
  
  `status` VARCHAR(20) NOT NULL COMMENT '状态: pending, waiting, running, success, failed, skipped, cancelled',
  
  `dependencies` JSON COMMENT '依赖节点',
  `dependency_status` JSON COMMENT '依赖状态',
  
  `scheduled_at` DATETIME(3) COMMENT '计划执行时间',
  `started_at` DATETIME(3) COMMENT '开始时间',
  `finished_at` DATETIME(3) COMMENT '完成时间',
  `duration_ms` BIGINT COMMENT '执行时长',
  `retry_count` INT DEFAULT 0 COMMENT '重试次数',
  
  `error_message` TEXT COMMENT '错误消息',
  `error_stack` TEXT COMMENT '错误堆栈',
  
  `input` JSON COMMENT '输入参数',
  `output` JSON COMMENT '输出结果',
  
  `condition` JSON COMMENT '执行条件',
  `condition_result` VARCHAR(20) COMMENT '条件结果',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_execution_node` (`execution_id`, `node_id`),
  KEY `idx_execution` (`execution_id`),
  KEY `idx_status` (`status`),
  KEY `idx_job_execution` (`job_execution_id`),
  KEY `idx_started_at` (`started_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作流节点执行表';

-- ============================================
-- 6. 系统管理模块
-- ============================================

-- 6.1 操作审计表
CREATE TABLE IF NOT EXISTS `sys_audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED COMMENT '用户ID',
  `username` VARCHAR(100) COMMENT '用户名',
  `action` VARCHAR(50) NOT NULL COMMENT '操作: login, logout, create, update, delete, enable, disable, run, stop',
  `resource_type` VARCHAR(50) NOT NULL COMMENT '资源类型: job, group, template, alert_rule, workflow, user',
  `resource_id` BIGINT UNSIGNED COMMENT '资源ID',
  `resource_name` VARCHAR(200) COMMENT '资源名称',
  
  `old_value` JSON COMMENT '修改前的值',
  `new_value` JSON COMMENT '修改后的值',
  `diff` TEXT COMMENT '差异(文本格式)',
  
  `ip` VARCHAR(50) COMMENT 'IP地址',
  `user_agent` VARCHAR(500) COMMENT 'User Agent',
  `request_id` VARCHAR(64) COMMENT '请求ID',
  
  `result` VARCHAR(20) DEFAULT 'success' COMMENT '结果: success, failed',
  `error_message` TEXT COMMENT '错误消息',
  `created_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_action` (`action`),
  KEY `idx_resource` (`resource_type`, `resource_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_request_id` (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作审计表';

-- 6.2 系统配置表
CREATE TABLE IF NOT EXISTS `sys_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `config_key` VARCHAR(100) NOT NULL COMMENT '配置键',
  `config_value` TEXT COMMENT '配置值',
  `value_type` VARCHAR(20) DEFAULT 'string' COMMENT '值类型: string, int, bool, json',
  `description` VARCHAR(500) COMMENT '描述',
  `category` VARCHAR(50) DEFAULT 'system' COMMENT '配置分类: system, alert, storage, notification',
  `is_encrypted` TINYINT(1) DEFAULT 0 COMMENT '是否加密',
  `is_public` TINYINT(1) DEFAULT 0 COMMENT '是否公开',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_key` (`config_key`),
  KEY `idx_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';

-- 6.3 系统通知表
CREATE TABLE IF NOT EXISTS `sys_notifications` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `type` VARCHAR(50) NOT NULL COMMENT '通知类型: job_failed, job_success, system_alert, etc.',
  `title` VARCHAR(200) NOT NULL COMMENT '标题',
  `content` TEXT COMMENT '内容',
  `link` VARCHAR(500) COMMENT '链接',
  
  `related_type` VARCHAR(50) COMMENT '关联类型: job, execution, alert',
  `related_id` BIGINT UNSIGNED COMMENT '关联ID',
  
  `read` TINYINT(1) DEFAULT 0 COMMENT '是否已读',
  `read_at` DATETIME(3) COMMENT '阅读时间',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  KEY `idx_user_read` (`user_id`, `read`),
  KEY `idx_related` (`related_type`, `related_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统通知表';

-- ============================================
-- 7. 模板系统模块
-- ============================================

-- 7.1 工作流模板表
CREATE TABLE IF NOT EXISTS `sys_workflow_templates` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `template_id` VARCHAR(64) NOT NULL COMMENT '模板ID',
  `name` VARCHAR(100) NOT NULL COMMENT '模板名称',
  `display_name` VARCHAR(200) NOT NULL COMMENT '显示名称',
  `description` TEXT COMMENT '描述',
  
  `dag` JSON NOT NULL COMMENT 'DAG结构（同工作流定义）',
  `params_schema` JSON COMMENT '全局参数Schema',
  `default_params` JSON COMMENT '默认全局参数',
  
  `sub_templates` JSON COMMENT '引用的原子任务模板列表',
  
  `icon` VARCHAR(50) COMMENT '图标',
  `category` VARCHAR(50) COMMENT '分类',
  `tags` JSON COMMENT '标签',
  
  `is_public` TINYINT(1) DEFAULT 1 COMMENT '是否公开',
  `version` VARCHAR(20) DEFAULT '1.0.0',
  `created_by` BIGINT UNSIGNED COMMENT '创建人',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_template_id` (`template_id`),
  KEY `idx_category` (`category`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作流模板表';

-- 7.2 组合任务模板表
CREATE TABLE IF NOT EXISTS `sys_composite_templates` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `template_id` VARCHAR(64) NOT NULL COMMENT '模板ID',
  `name` VARCHAR(100) NOT NULL COMMENT '模板名称',
  `display_name` VARCHAR(200) NOT NULL COMMENT '显示名称',
  `description` TEXT COMMENT '描述',
  
  `workflow_template_id` VARCHAR(64) NOT NULL COMMENT '工作流模板ID',
  
  `task_type` VARCHAR(50) DEFAULT 'composite' COMMENT '任务类型: composite',
  `trigger_config` JSON COMMENT '触发配置',
  
  `param_mapping` JSON COMMENT '参数映射',
  `exposed_params_schema` JSON COMMENT '对外暴露的参数Schema',
  
  `icon` VARCHAR(50) COMMENT '图标',
  `category` VARCHAR(50) COMMENT '分类',
  `tags` JSON COMMENT '标签',
  
  `is_public` TINYINT(1) DEFAULT 1 COMMENT '是否公开',
  `version` VARCHAR(20) DEFAULT '1.0.0',
  `created_by` BIGINT UNSIGNED COMMENT '创建人',
  
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_template_id` (`template_id`),
  KEY `idx_workflow` (`workflow_template_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='组合任务模板表';

-- ============================================
-- 插入默认数据
-- ============================================

-- 插入默认角色
INSERT INTO `roles` (`name`, `display_name`, `description`, `permissions`) VALUES
('admin', '管理员', '拥有所有权限', '["*"]'),
('operator', '操作员', '可以创建和执行任务', '["job:create", "job:update", "job:delete", "job:run", "job:read"]'),
('viewer', '查看者', '只能查看任务和日志', '["job:read", "execution:read", "log:read"]')
ON DUPLICATE KEY UPDATE `updated_at`=NOW();

-- 插入默认系统配置
INSERT INTO `sys_configs` (`config_key`, `config_value`, `value_type`, `description`, `category`, `is_public`) VALUES
('alert.default_channels', '["email"]', 'json', '默认告警渠道', 'alert', 1),
('alert.batch_size', '10', 'int', '告警批量发送数量', 'alert', 1),
('log.retention_days', '30', 'int', '日志保留天数', 'system', 1),
('execution.max_concurrent', '100', 'int', '最大并发执行数', 'system', 1),
('workflow.timeout', '3600', 'int', '工作流默认超时时间(秒)', 'system', 1)
ON DUPLICATE KEY UPDATE `updated_at`=NOW();