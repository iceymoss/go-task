package models

import (
	libDB "github.com/iceymoss/go-task/pkg/db"
)

// RegisterMySQLModels 注册所有MySQL模型
func RegisterMySQLModels() error {
	db := libDB.GetMysqlConn(libDB.MYSQL_DB_GO_TASK)
	// 用户和权限相关
	err := db.AutoMigrate(
		&User{},
		&Session{},
		&Role{},
		&UserRole{},
	)
	if err != nil {
		return err
	}

	// 任务相关
	err = db.AutoMigrate(
		&Job{},
		&JobGroup{},
		&JobVersion{},
		&ParamTemplate{},
	)
	if err != nil {
		return err
	}

	// 执行相关
	err = db.AutoMigrate(
		&JobExecution{},
		&JobLog{},
	)
	if err != nil {
		return err
	}

	// 告警相关
	err = db.AutoMigrate(
		&AlertRule{},
		&AlertChannel{},
		&AlertHistory{},
		&AlertSilence{},
	)
	if err != nil {
		return err
	}

	// 工作流相关
	err = db.AutoMigrate(
		&Workflow{},
		&WorkflowExecution{},
		&WorkflowNodeExecution{},
	)
	if err != nil {
		return err
	}

	// 模板相关
	err = db.AutoMigrate(
		&TaskTemplate{},
		&WorkflowTemplate{},
		&CompositeTemplate{},
	)
	if err != nil {
		return err
	}

	// 系统相关
	err = db.AutoMigrate(
		&AuditLog{},
		&Config{},
		&Notification{},
	)
	if err != nil {
		return err
	}

	return nil
}

// GetAllModels 获取所有模型列表
func GetAllModels() []interface{} {
	return []interface{}{
		// 用户和权限
		&User{},
		&Session{},
		&Role{},
		&UserRole{},

		// 任务
		&Job{},
		&JobGroup{},
		&JobVersion{},
		&ParamTemplate{},

		// 执行
		&JobExecution{},
		&JobLog{},

		// 告警
		&AlertRule{},
		&AlertChannel{},
		&AlertHistory{},
		&AlertSilence{},

		// 工作流
		&Workflow{},
		&WorkflowExecution{},
		&WorkflowNodeExecution{},

		// 模板
		&TaskTemplate{},
		&WorkflowTemplate{},
		&CompositeTemplate{},

		// 系统
		&AuditLog{},
		&Config{},
		&Notification{},
	}
}
