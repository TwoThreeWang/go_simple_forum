package handler

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/kingwrcy/hn/model"
	"github.com/kingwrcy/hn/utils"
	"github.com/kingwrcy/hn/vo"
	"github.com/samber/do"
	"github.com/spf13/cast"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserHandler struct {
	injector *do.Injector
	db       *gorm.DB
}

func NewUserHandler(injector *do.Injector) (*UserHandler, error) {
	return &UserHandler{
		injector: injector,
		db:       do.MustInvoke[*gorm.DB](injector),
	}, nil
}

func (u *UserHandler) Login(c *gin.Context) {
	var request vo.LoginRequest
	err := c.Bind(&request)
	if err != nil {
		c.HTML(200, "login.gohtml", gin.H{
			"msg":      "参数错误",
			"selected": "login",
		})
		return
	}
	// 验证 Turnstile 令牌
	if request.CfTurnstile != "" {
		remoteIP := c.ClientIP()
		_, err := utils.VerifyTurnstileToken(c, request.CfTurnstile, remoteIP)
		if err!= nil {
			c.HTML(200, "login.gohtml", gin.H{
				"msg":      "验证 Turnstile 令牌失败：" + err.Error(),
				"selected": "login",
			})
			return
		}
	}else{
		c.HTML(200, "login.gohtml", gin.H{
			"msg":      "验证 Turnstile 令牌失败：缺少验证参数",
			"selected": "login",
		})
		return
	}
	var user model.TbUser
	if err := u.db.
		Where("username = ?", request.Username).Or("email = ?", request.Username).
		First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {

		c.HTML(200, "login.gohtml", gin.H{
			"msg":      "登录失败，用户名或者密码不正确",
			"selected": "login",
		})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
		c.HTML(200, "login.gohtml", gin.H{
			"msg":      "登录失败，用户名或者密码不正确",
			"selected": "login",
		})
		return
	}
	if user.Status == "Banned" {
		c.HTML(200, "login.gohtml", gin.H{
			"msg":      "用户已被ban",
			"selected": "login",
		})
		return
	}

	cookieData := vo.Userinfo{
		Username: user.Username,
		Role:     user.Role,
		ID:       user.ID,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}
	refer := c.GetHeader("refer")
	if refer == "" {
		refer = "/"
	}
	c.Redirect(302, refer)
	session := sessions.Default(c)
	session.Set("login", true)
	session.Set("userinfo", cookieData)
	_ = session.Save()
	return
}

// Oauth 三方登录回调处理逻辑
func (u *UserHandler) Oauth(c *gin.Context) {
	refer := c.GetHeader("refer")
	if refer == "" {
		refer = "/"
	}
	userinfo := GetCurrentUser(c)
	inviteCode := c.DefaultPostForm("invite_code", "open")
	if inviteCode == "" {
		inviteCode = "open"
	}
	gCsrfToken := c.PostForm("g_csrf_token")
	if gCsrfToken == "" {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "参数错误：No CSRF token in post body.",
		}))
		return
	}
	CookiegCsrfToken, err := c.Request.Cookie("g_csrf_token")
	if err != nil || CookiegCsrfToken.Value != gCsrfToken {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "参数错误：Failed to verify double submit cookie.",
		}))
		return
	}
	clientID := os.Getenv("ClientID")
	//clientSecret := os.Getenv("ClientSecret")
	credential := c.PostForm("credential")
	data, err := idtoken.Validate(c, credential, clientID)
	if err != nil {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "Google 登陆失败，请稍后再试！",
		}))
		return
	}
	userInfo := data.Claims
	gid := userInfo["sub"].(string)
	username := userInfo["name"].(string)
	email := userInfo["email"].(string)
	avatar := userInfo["picture"].(string)
	var user model.TbUser
	// 如果用户已经登录，默认就是绑定三方账号
	if userinfo != nil {
		if err := u.db.Where("google_id = ?", gid).First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			updateData := map[string]interface{}{
				"google_id":  gid,
				"updated_at": time.Now(),
			}
			affected := u.db.Model(&model.TbUser{}).Where("id = ?", userinfo.ID).
				Updates(updateData)
			if affected.RowsAffected == 0 {
				// 没有记录被更新，可能是没有找到匹配的记录
				c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
					"title": "Success", "msg": "操作成功，但是没有内容被更新！",
				}))
				return
			}
			c.Redirect(302, refer)
			return
		} else {
			// 已经绑定了其他用户
			c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
				"title": "Error", "msg": "该账号已经绑定其他用户，用户名为：" + user.Username + "！",
			}))
			return
		}
	}

	// 先查一下用户是否已存在，如果存在直接登录，如果不存在新增用户并登录
Login:
	if err := u.db.Where("google_id = ?", gid).First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果用户不存在，先去注册
		user.Email = email
		user.Username = username
		user.Avatar = "/img_dl?url=" + avatar
		user.GoogleId = gid
		err := u.OauthRegister(c, user, inviteCode)
		if err != nil {
			c.HTML(200, "login.gohtml", gin.H{
				"msg":      err.Error(),
				"selected": "login",
			})
			return
		} else {
			// 注册成功，重新尝试登陆
			goto Login
		}
	}
	if user.Status == "Banned" {
		c.HTML(200, "login.gohtml", gin.H{
			"msg":      "登录失败，用户已被ban",
			"selected": "login",
		})
		return
	}
	cookieData := vo.Userinfo{
		Username: user.Username,
		Role:     user.Role,
		ID:       user.ID,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}
	c.Redirect(302, refer)
	session := sessions.Default(c)
	session.Set("login", true)
	session.Set("userinfo", cookieData)
	_ = session.Save()
	return
}

// OauthRegister 三方登录新用户注册流程
func (u *UserHandler) OauthRegister(c *gin.Context, user model.TbUser, code string) error {
	var settings model.TbSettings
	u.db.First(&settings)
	if settings.Content.RegMode == "shutdown" {
		return errors.New("本站目前未开放注册！")
	}

	var invited model.TbInviteRecord
	if settings.Content.RegMode == "invite" {
		err := u.db.Where("code = ? and status = 'ENABLE'", code).First(&invited).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("邀请码已使用/已过期/无效！")
		}
	}
	user.Status = "Active"
	user.CommentCount = 0
	user.PostCount = 0
	user.Bio = "这个人不懒, 但也没有介绍."
	user.Role = "0"
Save:
	err := u.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Save(&user).Error
		if err != nil {
			return err
		}
		if settings.Content.RegMode == "invite" {
			err = tx.Model(&invited).Where("id=?", invited.ID).Updates(model.TbInviteRecord{
				InvitedUserId: user.ID,
				InvalidAt:     time.Now(),
				Status:        "DISABLE",
			}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// 如果用户名重复了，添加尾缀后重试
		user.Username = user.Username + "_g"
		goto Save
	} else if err != nil {
		return errors.New("系统异常，新用户注册失败！")
	}
	return nil
}

func (u *UserHandler) ToLogin(c *gin.Context) {
	var settings model.TbSettings
	u.db.First(&settings)
	c.HTML(200, "login.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected": "login",
	}))
}
func (u *UserHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{Path: "/", MaxAge: -1})
	session.Save()
	c.Redirect(302, "/")
}

func (u *UserHandler) Asks(c *gin.Context) {
	userid := c.Param("userid")
	p := c.DefaultQuery("p", "1")
	page := cast.ToInt(p)
	size := 10

	var user model.TbUser
	if err := u.db.Preload(clause.Associations).Where("id= ?", userid).First(&user).Error; err == gorm.ErrRecordNotFound {
		c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "请核实用户是否存在.",
		}))
		return
	}

	var invitedUsername string
	u.db.Model(&model.TbInviteRecord{}).Select("username").Where("invitedUsername = ?", user.Username).First(&invitedUsername)

	var total int64
	var posts []model.TbPost

	tx := u.db.Model(&model.TbPost{}).Preload(clause.Associations).
		Where("user_id = ? and status ='Active' and type = 'ask'", user.ID)
	tx.Count(&total)
	tx.Order("created_at desc").Offset((cast.ToInt(page) - 1) * size).Limit(size).
		Find(&posts)
	totalPage := total / cast.ToInt64(size)

	if total%cast.ToInt64(size) > 0 {
		totalPage = totalPage + 1
	}

	c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected":        "mine",
		"user":            user,
		"sub":             "ask",
		"posts":           posts,
		"invitedUsername": invitedUsername,
		"totalPage":       totalPage,
		"total":           total,
		"hasNext":         cast.ToInt64(page) < totalPage,
		"hasPrev":         page > 1,
		"currentPage":     cast.ToInt(page),
	}))
}

func (u *UserHandler) Links(c *gin.Context) {
	p := c.DefaultQuery("p", "1")
	page := cast.ToInt(p)
	size := 10

	userid := c.Param("userid")
	// 尝试将 userid 字符串转换为 int64 类型
	_, err := strconv.ParseInt(userid, 10, 64)
	if err != nil {
		c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "请核实后重试.",
		}))
		return
	}
	var user model.TbUser
	if err := u.db.Preload(clause.Associations).Where("id= ?", userid).First(&user).Error; err == gorm.ErrRecordNotFound {
		c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "请核实后重试.",
		}))
		return
	}

	var invitedUsername string
	var inviteUserId uint
	u.db.Model(&model.TbInviteRecord{}).Select("user_id").Where("\"invited_user_id\" = ?", user.ID).First(&inviteUserId)
	u.db.Model(&model.TbUser{}).Select("username").Where("\"id\" = ?", inviteUserId).First(&invitedUsername)

	var total int64
	var posts []model.TbPost
	tx := u.db.Model(&model.TbPost{}).Preload(clause.Associations).
		Where("user_id = ? and status ='Active' and type = 'link'", user.ID)

	tx.Count(&total)
	tx.Order("created_at desc").Offset((cast.ToInt(page) - 1) * size).Limit(size).
		Find(&posts)

	totalPage := total / (cast.ToInt64(size))

	if total%(cast.ToInt64(size)) > 0 {
		totalPage = totalPage + 1
	}

	data := []byte(user.Email)
	hash := md5.Sum(data)
	EmailHash := hex.EncodeToString(hash[:])

	c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected":        "mine",
		"user":            user,
		"EmailHash":       EmailHash,
		"sub":             "link",
		"posts":           posts,
		"invitedUsername": invitedUsername,
		"inviteUserId":    inviteUserId,
		"totalPage":       totalPage,
		"total":           total,
		"hasNext":         cast.ToInt64(page) < totalPage,
		"hasPrev":         page > 1,
		"currentPage":     cast.ToInt(page),
	}))
}

// Collects 用户中心收藏列表
func (u *UserHandler) Collects(c *gin.Context) {
	userid := c.Param("userid")
	p := c.DefaultQuery("p", "1")
	page := cast.ToInt(p)
	size := 10

	var user model.TbUser
	if err := u.db.Preload(clause.Associations).Where("id= ?", userid).First(&user).Error; err == gorm.ErrRecordNotFound {
		c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "请核实用户是否存在.",
		}))
		return
	}

	var invitedUsername string
	u.db.Model(&model.TbInviteRecord{}).Select("username").Where("invitedUsername = ?", user.Username).First(&invitedUsername)

	var total int64
	var posts []model.TbPost

	tx := u.db.Model(&model.TbPost{}).Preload(clause.Associations).Where("status ='Active'")
	subQueryCollect := u.db.Table("tb_vote").Select("target_id").Where("tb_user_id = ? and type = 'POST' and action ='Collect'", userid)
	tx.Joins("INNER JOIN (?) AS vote_collect ON id = vote_collect.target_id", subQueryCollect)
	tx.Count(&total)
	tx.Order("created_at desc").Offset((cast.ToInt(page) - 1) * size).Limit(size).
		Find(&posts)
	totalPage := total / cast.ToInt64(size)

	if total%cast.ToInt64(size) > 0 {
		totalPage = totalPage + 1
	}

	c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected":        "mine",
		"user":            user,
		"sub":             "collects",
		"posts":           posts,
		"invitedUsername": invitedUsername,
		"totalPage":       totalPage,
		"total":           total,
		"hasNext":         cast.ToInt64(page) < totalPage,
		"hasPrev":         page > 1,
		"currentPage":     cast.ToInt(page),
	}))
}

func (u *UserHandler) UserEdit(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	userid := c.Param("userid")
	uid := fmt.Sprintf("%d", userinfo.ID)
	if uid != userid {
		c.HTML(200, "profiledit.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "不允许修改非本人信息！",
		}))
		return
	}
	var user model.TbUser
	if err := u.db.Preload(clause.Associations).Where("id= ?", userid).First(&user).Error; err == gorm.ErrRecordNotFound {
		c.HTML(200, "profiledit.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "请核实用户是否存在或禁用.",
		}))
		return
	}

	c.HTML(200, "profiledit.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected": "mine",
		"user":     user,
		"uid":      userinfo.ID,
		"sub":      "link",
	}))
}

func (u *UserHandler) SaveUser(c *gin.Context) {
	userinfo := GetCurrentUser(c)

	var user vo.EditUserRequest

	if err := c.Bind(&user); err != nil {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "内容异常，请检查后重试！",
		}))
		return
	}

	if userinfo.ID != user.Uid {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "确定【" + user.Username + "】是你本人？请核对用户名！",
		}))
		return
	}

	if len(user.Username) < 3 {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "用户名长度必须大于3位",
		}))
		return
	}
	if _, ok := mail.ParseAddress(user.Email); ok != nil {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "邮箱格式不正确",
		}))
		return
	}
	updateData := map[string]interface{}{
		"username":   user.Username,
		"avatar":     user.Avatar,
		"email":      user.Email,
		"bio":        user.Bio,
		"updated_at": time.Now(),
	}
	if user.Password != "" {
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
				"title": "Error", "msg": "系统错误，请稍后重试！",
			}))
			return
		}
		updateData = map[string]interface{}{
			"username":   user.Username,
			"avatar":     user.Avatar,
			"email":      user.Email,
			"bio":        user.Bio,
			"password":   string(hashedPwd),
			"updated_at": time.Now(),
		}
	}

	affected := u.db.Model(&model.TbUser{}).Where("id = ?", user.Uid).
		Updates(updateData)
	if affected.RowsAffected == 0 {
		// 没有记录被更新，可能是没有找到匹配的记录
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Success", "msg": "操作成功，但是没有内容被更新！",
		}))
		return
	}
	// 更新登录的cookie
	cookieData := vo.Userinfo{
		Username: user.Username,
		Role:     userinfo.Role,
		ID:       userinfo.ID,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}
	session := sessions.Default(c)
	session.Set("login", true)
	session.Set("userinfo", cookieData)
	_ = session.Save()
	c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"title": "Success", "msg": "用户信息修改成功",
	}))
	return
}

// SetStatus 设置用户状态
func (u *UserHandler) SetStatus(c *gin.Context) {
	uid := c.Query("id")
	key := c.Query("key")
	if uid == "" {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error", "msg": "参数错误！",
		}))
		return
	}
	userinfo := GetCurrentUser(c)
	updateData := map[string]interface{}{}
	where := "1=1"
	msg := ""
	if strings.Contains("ActiveBannedWait", key) && userinfo.Role == "admin" {
		updateData = map[string]interface{}{
			"status":     key,
			"updated_at": time.Now(),
		}
		msg = "操作成功！"
	} else {
		where = "MD5(email)='" + key + "'"
		updateData = map[string]interface{}{
			"status": "Active",
		}
		msg = "激活成功，欢迎加入！"
	}
	affected := u.db.Model(&model.TbUser{}).Where("id = ?", uid).Where(where).
		Updates(updateData)
	if affected.RowsAffected == 0 {
		// 没有记录被更新，可能是没有找到匹配的记录
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Success", "msg": "操作成功，但是没有内容被更新！",
		}))
		return
	}
	c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"title": "Success", "msg": msg,
	}))
	return
}

func (u *UserHandler) ToMessage(c *gin.Context) {

	var messages []model.TbMessage
	var total int64
	userinfo := GetCurrentUser(c)
	page := cast.ToInt(c.DefaultQuery("p", "1"))
	size := 25

	u.db.Where("to_user_id = ?", userinfo.ID).Count(&total)
	u.db.Where("to_user_id = ?", userinfo.ID).Limit(size).Offset((page - 1) * size).
		Order("created_at desc").Find(&messages)

	c.HTML(200, "message.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected": "message",
		"messages": messages,
		"total":    total,
	}))
}

// InviteList 用户邀请码列表
func (u *UserHandler) InviteList(c *gin.Context) {

	var invites []model.TbInviteRecord
	var total int64
	userinfo := GetCurrentUser(c)
	page := cast.ToInt(c.DefaultQuery("p", "1"))
	size := 25

	u.db.Model(&model.TbInviteRecord{}).Where("user_id = ?", userinfo.ID).Count(&total)
	u.db.Model(&model.TbInviteRecord{}).Where("user_id = ?", userinfo.ID).Limit(size).Offset((page - 1) * size).
		Order("created_at desc").Find(&invites)
	// 获取当前时间戳
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	c.HTML(200, "inviteCode.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected":  "invite",
		"invites":   invites,
		"total":     total,
		"timestamp": timestamp,
	}))
}

// InviteNew 新建一个邀请码
func (u *UserHandler) InviteNew(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	// 扣减积分
	var user model.TbUser
	u.db.Model(&model.TbUser{}).Where("id = ?", userinfo.ID).First(&user)
	if user.Points-50 < 0 {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error",
			"msg":   "积分不足，无法兑换新的邀请码！",
		}))
		return
	}
	user.Points = user.Points - 50
	// 生成新的邀请码
	inviteRecord := model.TbInviteRecord{
		UserId: userinfo.ID,
		Code:   RandStringBytesMaskImpr(10),
		Status: "ENABLE",
	}

	err := u.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Save(&user).Error
		if err != nil {
			return err
		}
		return tx.Save(&inviteRecord).Error
	})
	if err != nil {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "Error",
			"msg":   "系统错误，请稍后重试！",
		}))
		return
	}
	c.Redirect(301, "/u/invite")
}

func (u *UserHandler) SetAllRead(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}
	u.db.Model(&model.TbMessage{}).Where("to_user_id = ? and read = 'N'", userinfo.ID).Update("read", "Y")
	u.ToMessage(c)
}

func (u *UserHandler) SetSingleRead(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}

	if id, ok := c.GetQuery("id"); ok {
		log.Printf("get id %+v", id)
		u.db.Model(&model.TbMessage{}).Where("id = ? and to_user_id = ? and read = 'N'", id, userinfo.ID).Update("read", "Y")
	}
	u.ToMessage(c)
}

func (u *UserHandler) Comments(c *gin.Context) {
	p := c.DefaultQuery("p", "1")
	page := cast.ToInt(p)
	size := 10

	userid := c.Param("userid")
	var user model.TbUser
	if err := u.db.Preload(clause.Associations).Where("id= ?", userid).First(&user).Error; err == gorm.ErrRecordNotFound {
		c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "mine",
			"msg":      "请核实用户是否存在.",
		}))
		return
	}

	var invitedUsername string
	var total int64

	u.db.Model(&model.TbInviteRecord{}).Select("username").Where("\"invitedUsername\" = ?", user.Username).First(&invitedUsername)
	var comments []model.TbComment
	tx := u.db.Model(&model.TbComment{}).Preload("Post").
		Preload("User").
		Where("user_id = ? ", user.ID)
	tx.Count(&total)
	tx.Order("created_at desc").Offset((cast.ToInt(page) - 1) * size).Limit(size).
		Find(&comments)

	totalPage := total / (cast.ToInt64(size))

	if total%(cast.ToInt64(size)) > 0 {
		totalPage = totalPage + 1
	}

	c.HTML(200, "profile.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected":        "mine",
		"user":            user,
		"sub":             "comments",
		"comments":        comments,
		"invitedUsername": invitedUsername,
		"totalPage":       totalPage,
		"total":           total,
		"hasNext":         cast.ToInt64(page) < totalPage,
		"hasPrev":         page > 1,
		"currentPage":     cast.ToInt(page),
	}))
}

func (u *UserHandler) ToInvited(c *gin.Context) {
	var settings model.TbSettings

	u.db.First(&settings)

	if settings.Content.RegMode == "shutdown" {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "/",
		}))
		return
	}

	code := c.Param("code")
	if code == "" {
		c.Redirect(200, "/")
		return
	}
	if settings.Content.RegMode == "open" {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"selected": "/",
			"code":     code,
		}))
		return
	}
	var invited model.TbInviteRecord
	err := u.db.Where("code = ? and status = 'ENABLE'", code).First(&invited).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"codeIsInvalid": true,
			"msg":           "邀请码已使用/已过期/无效",
		}))
		return
	}
	var invitedUsername string
	u.db.Model(&model.TbUser{}).Select("username").Where("\"id\" = ?", invited.UserId).First(&invitedUsername)

	c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected":        "/",
		"invited":         invited,
		"invitedUsername": invitedUsername,
		"code":            code,
	}))
}

func (u *UserHandler) DoInvited(c *gin.Context) {
	var settings model.TbSettings
	u.db.First(&settings)

	if settings.Content.RegMode == "shutdown" {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"codeIsInvalid": true,
			"msg":           "目前不开放注册",
		}))
		return
	}

	code := c.Param("code")
	if code == "" {
		c.Redirect(200, "/")
		return
	}

	var invited model.TbInviteRecord
	var user model.TbUser
	if settings.Content.RegMode == "invite" {
		err := u.db.Where("code = ? and status = 'ENABLE'", code).First(&invited).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
				"codeIsInvalid": true,
				"msg":           "邀请码已使用/已过期/无效",
			}))
			return
		}
	}

	var request vo.RegisterRequest
	if err := c.Bind(&request); err != nil {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "参数无效", "code": code,
		}))
		return
	}
	// 验证 Turnstile 令牌
	if request.CfTurnstile != "" {
		remoteIP := c.ClientIP()
		_, err := utils.VerifyTurnstileToken(c, request.CfTurnstile, remoteIP)
		if err!= nil {
			c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
				"msg": "验证 Turnstile 令牌失败：" + err.Error(),
			}))
			return
		}
	}else{
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "验证 Turnstile 令牌失败：缺少验证参数",
		}))
		return
	}

	if len(request.Username) < 3 {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "用户名长度必须大于3位", "code": code,
		}))
		return
	}
	if len(request.Password) < 5 {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "密码长度必须大于5位", "code": code,
		}))
		return
	}
	if request.Password != request.RepeatPassword {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "两次密码不一致", "code": code,
		}))
		return
	}
	if _, ok := mail.ParseAddress(request.Email); ok != nil {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "邮箱格式不正确", "code": code,
		}))
		return
	}
	user.Username = request.Username
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "系统异常", "code": code,
		}))
		return
	}
	user.Password = string(hashedPwd)
	user.Bio = request.Bio
	user.Email = request.Email
	user.Status = "Wait"
	user.CommentCount = 0
	user.PostCount = 0
	user.Bio = "这个人不懒, 但也没有介绍."
	// 生成1-6之间的随机整数
	rand.Seed(time.Now().UnixNano())
	randomNumber := rand.Intn(6) + 1
	user.Avatar = fmt.Sprintf("/static/avatar/%d.jpg", randomNumber)

	var totalUsers int64
	u.db.Table("tb_user").Where("id <> 999999999").Count(&totalUsers)
	if totalUsers == 0 {
		user.Role = "admin"
	} else {
		user.Role = "0"
	}

	err = u.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Save(&user).Error
		if err != nil {
			return err
		}
		if settings.Content.RegMode == "invite" {
			err = tx.Model(&invited).Where("id=?", invited.ID).Updates(model.TbInviteRecord{
				InvitedUserId: user.ID,
				InvalidAt:     time.Now(),
				Status:        "DISABLE",
			}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg":  "用户名已经存在了,换一个吧",
			"code": code,
		}))
		return
	} else if err != nil {
		c.HTML(200, "toBeInvited.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "系统异常",
		}))
		return
	}
	// 注册成功发送激活邮件
	siteName := os.Getenv("SiteName")
	siteUrl := os.Getenv("SiteUrl")
	data := []byte(user.Email)
	hash := md5.Sum(data)
	EmailHash := hex.EncodeToString(hash[:])

	// 将激活邮件发送给用户
	content := fmt.Sprintf("您好，<br><br>收到此邮件是因为您在<b>竹林</b>网站上进行了注册，<br><br>请点击链接激活账号：%s/u/status?id=%d&key=%s", siteUrl, user.ID, EmailHash)
	msg := utils.Email{}.Send(user.Email, "["+siteName+"] 账户激活邮件", content)
	if msg != "Success" {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "系统异常", "msg": "注册成功，但是激活邮件发送异常，请稍后登录个人中心重试！",
		}))
		return
	}
	c.HTML(200, "login.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"msg": "注册成功,立即登录",
	}))
}

func (u *UserHandler) ToList(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.Redirect(302, "/")
		return
	}
	var users []model.TbUser
	u.db.Where("ID <> 999999999").Order("id desc").Find(&users)
	c.HTML(200, "users.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"selected": "users",
		"users":    users,
	}))

}

func (u *UserHandler) ChangePoints(uid uint, chengeType, points int) error {
	// chengeType 0:减积分 1:增加积分 2:签到
	var user model.TbUser
	u.db.Where("ID = ?", uid).Order("id desc").First(&user)
	if user.Username == "" {
		return errors.New("用户没找到")
	}
	if chengeType == 0 {
		user.Points = user.Points - points
	} else {
		user.Points = user.Points + points
	}
	if user.Points < 0 {
		user.Points = 0
	}
	if chengeType == 2 {
		// 如果是签到，判断今天是否已经签到过了
		// 获取当前时间的年、月、日
		nowYear, nowMonth, nowDay := time.Now().Date()
		// 获取目标时间的年、月、日
		tYear, tMonth, tDay := user.PunchAt.Date()
		// 判断年、月、日是否相同
		if nowYear == tYear && nowMonth == tMonth && nowDay == tDay {
			return errors.New("今天已经签到过了，请勿重复签到！")
		}
	}
	if user.Role != "admin" {
		user.Role = utils.GetUserLevel(user.Points)
	}
	user.PunchAt = time.Now()
	affected := u.db.Model(&model.TbUser{}).Where("id = ?", uid).
		Updates(user)
	if affected.RowsAffected == 0 {
		return errors.New("没有记录被更新")
	}
	return nil
}

// Punch 签到功能
func (u *UserHandler) Punch(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.HTML(200, "login.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"msg": "先登陆后才能签到",
		}))
		return
	}
	// 使用当前时间作为随机数种子
	rand.Seed(time.Now().UnixNano())
	// 生成 1-10 之间的随机整数
	randomNumber := rand.Intn(10) + 1 // rand.Intn(10) 生成 0-9 的随机数，+1 使其变为 1-10
	err := u.ChangePoints(userinfo.ID, 2, randomNumber)
	if err != nil {
		c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
			"title": "请勿重复签到",
			"msg":   err.Error(),
		}))
		return
	}
	c.HTML(200, "result.gohtml", OutputCommonSession(u.injector, c, gin.H{
		"title": "Success",
		"msg":   fmt.Sprintf("签到成功，获得 %d 个竹笋！", randomNumber),
	}))
}
