package handler

import (
	"fmt"
	"go_simple_forum/model"
	"go_simple_forum/vo"
	"log"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

type InspectHandler struct {
	injector *do.Injector
	db       *gorm.DB
}

func newInspectHandler(injector *do.Injector) (*InspectHandler, error) {
	return &InspectHandler{
		injector: injector,
		db:       do.MustInvoke[*gorm.DB](injector),
	}, nil
}

func (p InspectHandler) Inspect(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil || (userinfo.Role != "admin" && userinfo.Role != "inspector") {
		c.JSON(200, gin.H{
			"msg": "非法访问",
		})
		return
	}
	uid := userinfo.ID
	var request vo.InspectRequest
	var inspectLog model.TbInspectLog
	if err := c.Bind(&request); err != nil {
		c.JSON(200, gin.H{
			"msg": "参数错误",
		})
		return
	}
	log.Printf("%+v", request)
	if request.PostID == 0 && request.CommentID == 0 {
		c.JSON(200, gin.H{
			"msg": "参数错误",
		})
		return
	}
	status := "Active"

	inspectLog.InspectType = request.InspectType
	inspectLog.PostID = request.PostID
	inspectLog.Reason = request.Reason
	inspectLog.Result = request.Result
	if request.Result == "reject" {
		inspectLog.Action = "deleted " + request.InspectType
		status = "Rejected"
	}

	inspectLog.InspectorID = uid
	var postUid uint
	var post model.TbPost
	if request.PostID > 0 {
		if err := p.db.Model(&model.TbPost{}).Where("id = ?", request.PostID).First(&post).Error; err == nil {
			inspectLog.Title = "链接:" + post.Title
			postUid = post.UserID
		}
	}

	var message model.TbMessage
	if request.PostID > 0 {
		message.FromUserID = 999999999
		message.CreatedAt = time.Now()
		message.UpdatedAt = time.Now()
		message.Read = "N"
		message.ToUserID = postUid
		message.Content = fmt.Sprintf("你的帖子审核通过啦 (<a class='bLink' href='/p/%s'>%s</a>)",
			post.Pid, post.Title)
		if request.Result == "reject" {
			message.Content = fmt.Sprintf("你的帖子被管理员删除 (<a class='bLink' href='/p/%s'>%s</a>)",
				post.Pid, post.Title)
		}
	}
	err := p.db.Transaction(func(tx *gorm.DB) error {
		if request.Result == "reject" {
			err := tx.Save(&inspectLog).Error
			if err != nil {
				return err
			}
		}
		// 修改帖子状态
		if request.PostID > 0 {
			err := tx.Model(model.TbPost{}).Where("id = ?", request.PostID).Update("status", status).Error
			if err != nil {
				return err
			}
			if err := tx.Save(&message).Error; err != nil {
				return err
			}
		}
		handler := UserHandler{p.injector, p.db}
		// 删除帖子要扣除积分
		if postUid > 0 && request.Result == "reject" {
			err := handler.ChangePoints(postUid, 0, 5)
			if err != nil {
				return err
			}
		}
		// 新审核通过的帖子要增加随机积分
		if postUid > 0 && request.Result == "pass" {
			// 使用当前时间作为随机数种子
			rand.Seed(time.Now().UnixNano())
			// 生成 1-10 之间的随机整数
			points := rand.Intn(5) + 1 // rand.Intn(10) 生成 0-9 的随机数，+1 使其变为 1-10
			err := handler.ChangePoints(postUid, 1, points)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(200, gin.H{
			"msg": err.Error(),
		})
		return
	}
	c.Redirect(302, "/wait")
	return
}
