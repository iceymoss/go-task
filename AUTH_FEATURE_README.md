# 认证与仪表盘功能说明

## 🎉 功能概述

本次更新为 Go-Task 平台添加了完整的用户认证系统和增强的 Web 仪表盘。

## ✨ 新增功能

### 1. 用户认证系统
- ✅ 用户登录/登出
- ✅ JWT Token 认证
- ✅ 会话管理
- ✅ 自动创建默认管理员账号
- ✅ 密码加密（bcrypt）

### 2. 增强 Web 仪表盘
- ✅ 精美的登录页面
- ✅ 实时统计卡片（总任务数、运行中、成功、失败）
- ✅ 数据可视化图表（Chart.js）
  - 任务状态分布饼图
  - 任务执行趋势折线图
- ✅ 任务列表（实时更新）
- ✅ 手动触发任务功能
- ✅ 用户信息显示
- ✅ 退出登录功能

### 3. 安全性
- ✅ 所有 API 需要认证（除登录接口）
- ✅ JWT Token 自动验证
- ✅ Token 过期处理
- ✅ 密码加密存储

## 🚀 快速开始

### 1. 启动服务

```bash
go run cmd/scheduler/main.go
```

服务将运行在 `http://localhost:9099`

### 2. 访问登录页面

在浏览器中打开：
```
http://localhost:9099/login.html
```

### 3. 使用默认账号登录

```
用户名: admin
密码: admin123
```

### 4. 访问仪表盘

登录成功后，会自动跳转到仪表盘：
```
http://localhost:9099/index.html
```

## 🔧 配置说明

### 配置文件 (configs/config.yaml)

```yaml
auth:
  jwt_secret: "your-secret-key-change-this-in-production"  # JWT 密钥（生产环境请修改）
  token_expire_hrs: 24  # Token 有效期（小时）
  default_admin:
    username: "admin"  # 默认管理员用户名
    password: "admin123"  # 默认管理员密码
    email: "admin@example.com"  # 默认管理员邮箱
```

### 生产环境注意事项

1. **修改 JWT 密钥**
   ```yaml
   jwt_secret: "your-very-secure-random-secret-key"
   ```

2. **修改默认管理员密码**
   - 首次登录后，立即修改密码
   - 或在配置文件中设置强密码

3. **启用 HTTPS**
   - 使用 Nginx/Apache 反向代理
   - 配置 SSL 证书

## 📊 API 接口

### 认证相关（无需 Token）

#### 登录
```
POST /api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    ...
  }
}
```

#### 刷新 Token
```
POST /api/auth/refresh
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}

Response:
{
  "token": "new-jwt-token..."
}
```

### 用户相关（需要 Token）

#### 获取当前用户信息
```
GET /api/auth/me
Authorization: Bearer <token>

Response:
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "role": "admin",
  ...
}
```

#### 登出
```
POST /api/auth/logout
Authorization: Bearer <token>

Response:
{
  "message": "logged out successfully"
}
```

### 仪表盘相关（需要 Token）

#### 获取统计数据
```
GET /api/dashboard/stats
Authorization: Bearer <token>

Response:
{
  "total_tasks": 3,
  "running_tasks": 1,
  "success_tasks": 1,
  "error_tasks": 1,
  "tasks": [...]
}
```

#### 获取任务列表
```
GET /api/tasks
Authorization: Bearer <token>

Response:
{
  "data": [...]
}
```

#### 手动触发任务
```
POST /api/tasks/:name/run
Authorization: Bearer <token>

Response:
{
  "message": "Triggered"
}
```

## 🗄️ 数据库表结构

### users 表
```sql
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `email` varchar(100) DEFAULT NULL,
  `role` varchar(20) DEFAULT 'user',
  `is_active` tinyint(1) DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `last_login_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### sessions 表
```sql
CREATE TABLE `sessions` (
  `id` varchar(36) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `token` varchar(500) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sessions_token` (`token`),
  KEY `idx_sessions_user_id` (`user_id`),
  KEY `idx_sessions_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 🔒 安全建议

1. **生产环境必须修改 JWT 密钥**
2. **使用强密码**
3. **启用 HTTPS**
4. **定期更新依赖**
5. **限制登录尝试次数**（可添加）
6. **启用审计日志**（可添加）

## 🐛 故障排除

### 问题1：无法连接数据库
```
❌ Failed to connect database
```
**解决：** 检查 `configs/config.yaml` 中的 MySQL 配置

### 问题2：登录失败
```
invalid username or password
```
**解决：**
- 确认使用正确的用户名和密码
- 检查数据库中是否存在默认管理员
- 首次启动会自动创建，如未创建请检查日志

### 问题3：Token 过期
```
Invalid or expired token
```
**解决：**
- 重新登录获取新 Token
- 或使用 `/api/auth/refresh` 刷新 Token

### 问题4：401 未授权错误
```
Unauthorized
```
**解决：**
- 确保在请求头中包含 `Authorization: Bearer <token>`
- 检查 Token 是否有效

## 📝 未来改进计划

- [ ] 添加用户注册功能
- [ ] 添加密码重置功能
- [ ] 添加多角色权限管理
- [ ] 添加审计日志
- [ ] 添加登录限制（防暴力破解）
- [ ] 添加双因素认证（2FA）
- [ ] 添加任务历史记录查看
- [ ] 添加任务配置编辑功能
- [ ] 添加实时 WebSocket 推送
- [ ] 添加更多图表和统计指标

## 📞 支持

如有问题，请：
1. 查看本文档的故障排除部分
2. 检查服务器日志
3. 提交 Issue

---

**祝使用愉快！🎉**