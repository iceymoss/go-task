package sql

import (
	"context"
	"fmt"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks/base_task"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const TaskName = "sql:sql"

// SqlTask SQL 查询任务
type SqlTask struct {
	base_task.BaseTask
}

func NewSqlTask() core.Task {
	return &SqlTask{
		BaseTask: base_task.BaseTask{
			Name:     TaskName,
			TaskType: constants.TaskTypeAPI,
		},
	}
}

// SqlParams 参数结构
type SqlParams struct {
	Query    string `json:"query" binding:"required"`
	Database string `json:"database"` // mysql 或其他数据库类型
}

func (t *SqlTask) Run(ctx context.Context, params map[string]any) error {
	// 解析参数
	p := parseParams(params)
	if p.Query == "" {
		return fmt.Errorf("query is required")
	}

	logger.Info("🚀 [SqlTask] Executing SQL query",
		zap.String("database", p.Database),
		zap.String("query", p.Query),
	)

	// 获取数据库连接
	var dbConn *gorm.DB
	switch p.Database {
	case "mysql", "":
		dbConn = db.GetMysqlConn(db.MYSQL_DB_GO_TASK)
	case "mongo":
		// MongoDB 支持（如果需要）
		return fmt.Errorf("mongodb not supported yet")
	default:
		return fmt.Errorf("unsupported database: %s", p.Database)
	}

	if dbConn == nil {
		return fmt.Errorf("database connection not found")
	}

	// 执行 SQL
	result := dbConn.Exec(p.Query)
	if result.Error != nil {
		logger.Error("❌ [SqlTask] Query failed",
			zap.String("query", p.Query),
			zap.Error(result.Error),
		)
		return fmt.Errorf("query failed: %w", result.Error)
	}

	// 获取影响行数
	rowsAffected := result.RowsAffected

	logger.Info("✅ [SqlTask] Query completed",
		zap.String("query", p.Query),
		zap.Int64("rows_affected", rowsAffected),
	)

	return nil
}

func parseParams(params map[string]any) SqlParams {
	p := SqlParams{
		Database: "mysql", // 默认 MySQL
	}

	if v, ok := params["query"].(string); ok {
		p.Query = v
	}
	if v, ok := params["database"].(string); ok {
		p.Database = v
	}

	return p
}
