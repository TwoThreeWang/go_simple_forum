package model

import (
	"gorm.io/gorm"
	"time"
)

const TableNameTbInviteRecord = "tb_invite_record"

type TbInviteRecord struct {
	gorm.Model
	UserId        uint      `gorm:"column:user_id;type:int;"`
	Code          string    `gorm:"column:code;type:varchar(30);unique"`
	InvitedUserId uint      `gorm:"column:invited_user_id;type:int"`
	InvalidAt     time.Time `gorm:"column:invalid_at;type:timestamp"`
	Status        string    `gorm:"column:status;type:varchar(20)"`
}

func (*TbInviteRecord) TableName() string {
	return TableNameTbInviteRecord
}
