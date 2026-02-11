package repo

import (
	"context"

	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/objects"
)

type JobRepo struct{}

func NewJobRepo() *JobRepo { return &JobRepo{} }

// GetActiveJobs 获取所有开启的任务
func (r *JobRepo) GetActiveJobs(ctx context.Context) ([]*objects.SysJob, error) {
	var list []*objects.SysJob
	err := db.GetMysqlConn(db.MYSQL_DB_GO_TASK).WithContext(ctx).Where("status = ?", 1).Find(&list).Error
	return list, err
}

// CreateLog 开始记录日志
func (r *JobRepo) CreateLog(ctx context.Context, log *objects.SysJobLog) error {
	return db.GetMysqlConn(db.MYSQL_DB_GO_TASK).Create(log).Error
}

// UpdateLog 任务结束更新日志
func (r *JobRepo) UpdateLog(ctx context.Context, log *objects.SysJobLog) error {
	return db.GetMysqlConn(db.MYSQL_DB_GO_TASK).WithContext(ctx).Save(log).Error
}

// UpdateJobParams UpdateJobCursor 更新任务的 Params (游标)
// 比如同步数据任务，跑完后把 {"last_id": 100} 更新回数据库
func (r *JobRepo) UpdateJobParams(ctx context.Context, jobID uint, newParams string) error {
	return db.GetMysqlConn(db.MYSQL_DB_GO_TASK).WithContext(ctx).Model(&objects.SysJob{}).
		Where("id = ?", jobID).Update("params", newParams).Error
}
