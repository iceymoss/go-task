#!/bin/bash

# Go-Task 认证功能测试脚本
# 用于快速验证认证和仪表盘功能

echo "=========================================="
echo "  Go-Task 认证功能测试脚本"
echo "=========================================="
echo ""

# 服务器地址
SERVER="http://localhost:9099"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_api() {
    local test_name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local token=$5
    
    echo -e "${YELLOW}测试: ${test_name}${NC}"
    echo "请求: ${method} ${endpoint}"
    
    if [ -z "$token" ]; then
        response=$(curl -s -X ${method} \
            -H "Content-Type: application/json" \
            -d "${data}" \
            "${SERVER}${endpoint}")
    else
        response=$(curl -s -X ${method} \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${token}" \
            -d "${data}" \
            "${SERVER}${endpoint}")
    fi
    
    echo "响应: ${response}"
    echo ""
}

# 1. 测试登录
echo "=========================================="
echo "步骤 1: 测试登录"
echo "=========================================="
login_response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "${SERVER}/api/auth/login")

echo "登录响应: ${login_response}"
echo ""

# 提取 token
TOKEN=$(echo ${login_response} | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ 登录失败，无法获取 token${NC}"
    echo "请确保服务器正在运行并且已创建默认管理员账号"
    exit 1
fi

echo -e "${GREEN}✅ 登录成功！Token: ${TOKEN:0:50}...${NC}"
echo ""

# 2. 测试获取用户信息
echo "=========================================="
echo "步骤 2: 测试获取当前用户信息"
echo "=========================================="
test_api "获取用户信息" "GET" "/api/auth/me" "" "${TOKEN}"

# 3. 测试获取仪表盘统计数据
echo "=========================================="
echo "步骤 3: 测试获取仪表盘统计数据"
echo "=========================================="
test_api "获取仪表盘统计" "GET" "/api/dashboard/stats" "" "${TOKEN}"

# 4. 测试获取任务列表
echo "=========================================="
echo "步骤 4: 测试获取任务列表"
echo "=========================================="
test_api "获取任务列表" "GET" "/api/tasks" "" "${TOKEN}"

# 5. 测试未授权访问（不带 token）
echo "=========================================="
echo "步骤 5: 测试未授权访问（安全测试）"
echo "=========================================="
echo -e "${YELLOW}测试: 访问需要认证的接口（不带 token）${NC}"
echo "请求: GET /api/tasks"
response=$(curl -s -X GET "${SERVER}/api/tasks")
echo "响应: ${response}"

if echo "$response" | grep -q "Unauthorized"; then
    echo -e "${GREEN}✅ 安全测试通过：未授权访问被拒绝${NC}"
else
    echo -e "${RED}❌ 安全测试失败：应该拒绝未授权访问${NC}"
fi
echo ""

# 6. 测试无效 token
echo "=========================================="
echo "步骤 6: 测试无效 token（安全测试）"
echo "=========================================="
echo -e "${YELLOW}测试: 访问需要认证的接口（无效 token）${NC}"
echo "请求: GET /api/auth/me"
response=$(curl -s -X GET \
    -H "Authorization: Bearer invalid-token-12345" \
    "${SERVER}/api/auth/me")
echo "响应: ${response}"

if echo "$response" | grep -q "Invalid or expired token"; then
    echo -e "${GREEN}✅ 安全测试通过：无效 token 被拒绝${NC}"
else
    echo -e "${RED}❌ 安全测试失败：应该拒绝无效 token${NC}"
fi
echo ""

# 7. 测试刷新 token
echo "=========================================="
echo "步骤 7: 测试刷新 token"
echo "=========================================="
test_api "刷新 token" "POST" "/api/auth/refresh" "{\"token\":\"${TOKEN}\"}" ""

# 8. 测试登出
echo "=========================================="
echo "步骤 8: 测试登出"
echo "=========================================="
test_api "登出" "POST" "/api/auth/logout" "" "${TOKEN}"

echo "=========================================="
echo -e "${GREEN}✅ 所有测试完成！${NC}"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 在浏览器中访问: ${SERVER}/login.html"
echo "2. 使用账号: admin / admin123 登录"
echo "3. 查看仪表盘功能"
echo ""
echo "详细文档请查看: AUTH_FEATURE_README.md"
echo ""