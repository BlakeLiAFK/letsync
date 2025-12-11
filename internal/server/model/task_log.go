package model

import (
	"time"
)

// TaskLog 任务日志表，用于记录证书申请/续期的详细过程
type TaskLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"size:64;index;not null" json:"task_id"`     // 任务唯一标识（UUID）
	CertID    uint      `gorm:"index;not null" json:"cert_id"`         // 关联的证书ID
	TaskType  string    `gorm:"size:50;not null" json:"task_type"`      // 任务类型：issue/renew
	Level     string    `gorm:"size:10;not null;default:'info'" json:"level"` // 日志级别：info/warn/error
	Message   string    `gorm:"type:text;not null" json:"message"`     // 日志消息
	CreatedAt time.Time `json:"created_at"`
}

// TaskLogStatus 任务状态跟踪表
type TaskLogStatus struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"size:64;uniqueIndex;not null" json:"task_id"` // 任务唯一标识（UUID）
	CertID    uint      `gorm:"index;not null" json:"cert_id"`        // 关联的证书ID
	TaskType  string    `gorm:"size:50;not null" json:"task_type"`      // 任务类型：issue/renew
	Status    string    `gorm:"size:20;not null;default:'running'" json:"status"` // 任务状态：running/completed/failed
	StartTime time.Time `gorm:"not null" json:"start_time"`           // 任务开始时间
	EndTime   *time.Time `json:"end_time"`                              // 任务结束时间
	CreatedAt time.Time `json:"created_at"`
}