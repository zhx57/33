package model

const TableNameTcWeltolkAutoreplyTasks = "tc_weltolk_autoreply_tasks"

// TcWeltolkAutoreplyTasks mapped from table <tc_weltolk_autoreply_tasks>
type TcWeltolkAutoreplyTasks struct {
	ID               uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	UID              int    `gorm:"not null;default:0" json:"uid"`
	Pid              int    `gorm:"not null;default:0" json:"pid"`
	Fname            string `gorm:"type:text;not null" json:"fname"`
	Tid              int64  `gorm:"not null;default:0" json:"tid"`
	LastFloor        int    `gorm:"not null;default:0" json:"last_floor"`
	LastRepliedPid   int64  `gorm:"not null;default:0" json:"last_replied_pid"`
	LastReplyTime    int    `gorm:"not null;default:0" json:"last_reply_time"`
	LastStatus       string `gorm:"type:varchar(32);default:''" json:"last_status"`
	LastError        string `gorm:"type:text" json:"last_error"`
	LastCheckTime    int    `gorm:"not null;default:0" json:"last_check_time"`
	Log              string `gorm:"type:longtext" json:"log"`
	ReplyContent     string `gorm:"type:text;not null" json:"reply_content"`
	ReplyInterval    int    `gorm:"not null;default:300" json:"reply_interval"`
	ReplyProbability int    `gorm:"not null;default:100" json:"reply_probability"`
	Enabled          int8   `gorm:"not null;default:1" json:"enabled"`
	RetryCount       int    `gorm:"not null;default:0" json:"retry_count"`
	TriggerMode      string `gorm:"type:varchar(20);not null;default:'new_floor'" json:"trigger_mode"`
	ReplyTarget      string `gorm:"type:varchar(20);not null;default:'floor'" json:"reply_target"`
	AllowReplied     int8   `gorm:"not null;default:0" json:"allow_replied"`
	MatchKeywords    string `gorm:"type:text" json:"match_keywords"`
}

// TableName TcWeltolkAutoreplyTasks's table name
func (*TcWeltolkAutoreplyTasks) TableName() string {
	return TableNameTcWeltolkAutoreplyTasks
}
