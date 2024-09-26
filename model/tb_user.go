package model

import (
	"gorm.io/gorm"
	"time"
)

const TableNameTbUser = "tb_user"

type TbUser struct {
	gorm.Model
	Username        string      `gorm:"column:username;type:varchar(30);unique"`
	Password        string      `gorm:"column:password;type:varchar(100)"`
	Role            string      `gorm:"column:role;type:varchar(20)"`
	Email           string      `gorm:"column:email;type:varchar(100)"`
	Bio             string      `gorm:"column:bio;type:varchar(100)"`
	CommentCount    int         `gorm:"column:commentCount;type:int"`
	PostCount       int         `gorm:"column:postCount;type:int"`
	Status          string      `gorm:"column:status;type:varchar(20)"`
	Avatar          string      `gorm:"column:avatar;type:varchar(100)"`
	Posts           []TbPost    `gorm:"foreignKey:UserID"`
	UpVotedPosts    []TbPost    `gorm:"many2many:tb_vote;"`
	Points          int         `gorm:"column:points;type:int;default:0"`
	PunchAt         time.Time   `gorm:"column:punch_at;type:timestamptz(6)"`
	Comments        []TbComment `gorm:"foreignKey:UserID"`
	UpVotedComments []TbComment `gorm:"many2many:tb_vote;"`
	GoogleId        string      `gorm:"column:google_id;type:varchar(100)"`
}

func (*TbUser) TableName() string {
	return TableNameTbUser
}
