package handler

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/feeds"
	"github.com/kingwrcy/hn/model"
	"github.com/kingwrcy/hn/utils"
	"github.com/kingwrcy/hn/vo"
	"github.com/russross/blackfriday"
	"github.com/samber/do"
	"github.com/snabb/sitemap"
	"github.com/spf13/cast"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type IndexHandler struct {
	injector *do.Injector
	db       *gorm.DB
}

func NewIndexHandler(injector *do.Injector) (*IndexHandler, error) {
	return &IndexHandler{
		injector: injector,
		db:       do.MustInvoke[*gorm.DB](injector),
	}, nil
}

func (i *IndexHandler) Index(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	begin := time.Now().AddDate(0, 0, -60)
	page := c.DefaultQuery("p", "1")
	topics := QueryPosts(i.db, vo.QueryPostsRequest{
		Userinfo:  userinfo,
		Begin:     &begin,
		OrderType: "index",
		Page:      cast.ToInt64(page),
		Size:      25,
	})
	if list, ok := topics["posts"].([]model.TbPost); ok && len(list) == 0 {
		c.Redirect(301, "/history")
		return
	}

	c.HTML(200, "index.html", OutputCommonSession(i.injector, c, gin.H{
		"selected": "/",
		"title":    "🔥热议",
		"slogan":   "链接有趣内容，聚合真实想法，和真实的人一起筛内容，不靠算法也能刷到好东西。",
	}, topics))
}

func (i *IndexHandler) SiteMap(c *gin.Context) {
	var items []model.TbPost
	i.db.Model(&model.TbPost{}).Where("status = 'Active'").Order("created_at desc").Find(&items)
	sm := sitemap.New()
	SiteUrl := os.Getenv("SiteUrl")
	for _, item := range items {
		t := item.Model.CreatedAt
		sm.Add(&sitemap.URL{
			Loc:        SiteUrl + "/p/" + item.Pid,
			LastMod:    &t,
			ChangeFreq: sitemap.Daily,
			Priority:   0.5,
		})
	}
	// 设置 Header，指示浏览器这是一个 XML 文件
	c.Header("Content-Type", "application/xml")
	// 将 Sitemap 内容写入响应
	_, err := sm.WriteTo(c.Writer)
	if err != nil {
		fmt.Println("Error writing sitemap:", err)
		c.String(500, "Error generating sitemap")
		return
	}
}

func (i *IndexHandler) Feed(c *gin.Context) {
	// 从 Gin 上下文中获取缓存实例
	cache := c.MustGet("cache").(*utils.Cache)
	// 从缓存中获取 items
	var items []model.TbPost
	postItems, ok := cache.Get("feedPostItems")
getPost:
	if !ok {
		// 查询帖子
		userinfo := GetCurrentUser(c)
		result := QueryPosts(i.db, vo.QueryPostsRequest{
			Userinfo:  userinfo,
			Page:      1,
			Size:      50,
			OrderType: "rss",
		})
		items = result["posts"].([]model.TbPost)
		// 将 items 放入缓存
		cache.Set("feedPostItems", items, 30*time.Minute)
	} else {
		// 使用缓存中的数据
		items, ok = postItems.([]model.TbPost)
		if !ok {
			goto getPost
		}
	}
	// 创建 RSS Feed 数据
	SiteUrl := os.Getenv("SiteUrl")
	rssFeed := &feeds.Feed{
		Title:       os.Getenv("SiteName"),
		Link:        &feeds.Link{Href: SiteUrl},
		Description: "竹林是一个链接优质内容和真实用户讨论的去算法推荐社区，由用户分享推荐优质资讯，聚焦真实评论与用户共鸣，和真实的人一起筛内容，依靠用户共识挑出值得一读的内容，不靠算法也能刷到好东西。",
		Created:     time.Now(),
		Updated:     time.Now(),
	}
	for _, item := range items {
		t := item.Model.CreatedAt
		description := item.Content
		for _, tag := range item.Tags {
			if tag.OpenShow >= 0 {
				description = "游客无法查看隐藏标签下的内容，请点击标题登录网站浏览！"
				break
			}
		}
		//if utf8.RuneCountInString(item.Content) > 250 {
		//	// 截取前100位
		//	description = string([]rune(item.Content)[:200]) + "..."
		//}
		// 使用 blackfriday 库将 Markdown 转换为 HTML
		description = string(blackfriday.MarkdownCommon([]byte(description)))
		itemUrl := SiteUrl + "/p/" + item.Pid
		content := description + "<br><br><b><a href=\"" + itemUrl + "\">评论也是内容的一部分，点击标题阅读完整话题和讨论</a></b>"
		feedItem := feeds.Item{
			Id:          item.Pid,
			IsPermaLink: "false",
			Title:       item.Title,
			Author:      &feeds.Author{Name: item.User.Username},
			Description: description,
			Content:     content,
			Link:        &feeds.Link{Href: itemUrl},
			Created:     t,
		}
		rssFeed.Items = append(rssFeed.Items, &feedItem)
	}
	// 生成 RSS XML 内容
	rssXml, err := rssFeed.ToRss()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// 设置响应头
	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	// 输出 RSS XML 内容
	c.String(200, rssXml)
}

func (i *IndexHandler) ToSearch(c *gin.Context) {
	c.HTML(200, "search.html", OutputCommonSession(i.injector, c, gin.H{
		"selected": "search",
	}))
}

func (i *IndexHandler) DoSearch(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	var request vo.QueryPostsRequest
	c.Bind(&request)
	request.Size = 25
	request.Userinfo = userinfo
	if request.Page <= 0 {
		request.Page = 1
	}
	c.HTML(200, "search.html", OutputCommonSession(i.injector, c, gin.H{
		"selected":  "search",
		"condition": request,
	}, QueryPosts(i.db, request)))
}

func (i *IndexHandler) ToNew(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}
	msg := ""
	if userinfo.Role != "admin" {
		role, err := strconv.Atoi(userinfo.Role)
		if role < 2 || err != nil {
			msg = "注意：信任级别 LV.2 以下的用户发表帖子需要审核才可以展示！"
		}
	}
	var tags []model.TbTag
	i.db.Model(&model.TbTag{}).Preload("Parent").Where("parent_id is null").Preload("Children", func(db *gorm.DB) *gorm.DB {
		return db.Order("name") // 对子标签进行排序
	}).Order("name").Find(&tags)
	c.HTML(200, "new.html", OutputCommonSession(i.injector, c, gin.H{
		"selected": "new",
		"tags":     tags,
		"msg":      msg,
	}))
}

func (i *IndexHandler) ToPost(c *gin.Context) {
	c.HTML(200, "post.html", OutputCommonSession(i.injector, c, gin.H{}))
}
func (i *IndexHandler) ToResetPwd(c *gin.Context) {
	c.HTML(200, "resetPwd.html", OutputCommonSession(i.injector, c, gin.H{}))
}
func (i *IndexHandler) ToResetPwdEdit(c *gin.Context) {
	key := c.DefaultQuery("key", "")
	c.HTML(200, "resetPwdEdit.html", OutputCommonSession(i.injector, c, gin.H{
		"key": key,
	}))
}

// DoResetPwd 重置密码邮件发送函数
func (i *IndexHandler) DoResetPwd(c *gin.Context) {
	var data vo.ResetPwd
	if err := c.Bind(&data); err != nil {
		c.HTML(200, "resetPwd.html", OutputCommonSession(i.injector, c, gin.H{
			"msg": "内容异常，请检查后重试！",
		}))
		return
	}
	if data.Email == "" {
		c.HTML(200, "resetPwd.html", OutputCommonSession(i.injector, c, gin.H{
			"msg": "内容异常，请先输入注册邮箱！",
		}))
		return
	}
	// 校验邮箱是否存在
	var user model.TbUser
	if err := i.db.
		Where("email = ?", data.Email).
		First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {

		c.HTML(200, "resetPwd.html", gin.H{
			"msg": "内容异常，请确认注册邮箱是否正确！",
		})
		return
	}
	// 要编码的字符串
	message := string(user.ID) + "#," + user.Email
	// 将字符串编码为 Base64
	encodedMessage := base64.StdEncoding.EncodeToString([]byte(message))
	siteName := os.Getenv("SiteName")
	SiteUrl := os.Getenv("SiteUrl")
	// 将随机密码邮件发送给用户
	content := "您好，<br><br>收到此邮件是因为您在" + siteName + "网站上进行了重置密码的操作，<br><br>" +
		"请点击此链接重置密码：" + SiteUrl + "/resetPwdEdit?key=" + encodedMessage
	fmt.Println(content)
	msg := utils.Email{}.Send(data.Email, "["+siteName+"] 密码重置操作", content)
	if msg != "Success" {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "系统异常", "msg": "密码重置邮件发送异常，请稍后重试！",
		}))
		return
	}
	c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
		"title": "Success", "msg": "密码重置邮件已发送，请查收邮箱！",
	}))
}

// DoResetPwdEdit 重置密码操作函数
func (i *IndexHandler) DoResetPwdEdit(c *gin.Context) {
	var data vo.ResetPwd
	if err := c.Bind(&data); err != nil {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "Error", "msg": "内容异常，请检查后重试！",
		}))
		return
	}
	if data.Email == "" || data.Password == "" || data.Key == "" {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "Error", "msg": "内容异常，请检查后重试！",
		}))
		return
	}
	// 校验邮箱是否存在
	var user model.TbUser
	if err := i.db.
		Where("email = ?", data.Email).
		First(&user).Error; errors.Is(err, gorm.ErrRecordNotFound) {

		c.HTML(200, "result.html", gin.H{
			"title": "Error", "msg": "内容异常，请确认注册邮箱是否正确！",
		})
		return
	}
	// 验证key中的信息和实际信息
	decodedMessage, err := base64.StdEncoding.DecodeString(data.Key)
	if err != nil {
		c.HTML(200, "result.html", gin.H{
			"title": "Error", "msg": "内容异常，请检查后重试！",
		})
		return
	}
	keys := strings.Split(string(decodedMessage), "#,")
	if keys[0] != string(user.ID) || keys[1] != data.Email {
		c.HTML(200, "result.html", gin.H{
			"title": "Error", "msg": "内容异常，请检查后重试！",
		})
		return
	}
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "系统异常", "msg": "密码重置出错，请稍后重试！",
		}))
		return
	}
	affected := i.db.Model(&model.TbUser{}).Where("email = ?", data.Email).
		Updates(map[string]interface{}{
			"password":   string(hashedPwd),
			"updated_at": time.Now(),
		})
	if affected.RowsAffected == 0 {
		// 没有记录被更新，可能是没有找到匹配的记录
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "Error", "msg": "密码重置失败，请检查邮箱是否正确后重试！",
		}))
		return
	}
	c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
		"title": "密码重置成功", "msg": "现在点击右上角使用新密码登录！",
	}))
}

func (i *IndexHandler) ToAddTag(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.Redirect(302, "/tags")
		return
	}
	var parentTags []model.TbTag
	i.db.Find(&parentTags, "parent_id is null")

	c.HTML(200, "tagEdit.html", OutputCommonSession(i.injector, c, gin.H{
		"parents":  parentTags,
		"selected": "tags",
	}))
}
func (i *IndexHandler) ToEditTag(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.Redirect(302, "/tags")
		return
	}
	id := cast.ToString(c.Param("id"))
	var tag model.TbTag
	i.db.First(&tag, "id = ?", id)

	var parentTags []model.TbTag
	i.db.Find(&parentTags, "parent_id is null")

	c.HTML(200, "tagEdit.html", OutputCommonSession(i.injector, c, gin.H{
		"tag":      tag,
		"parentID": cast.ToInt(tag.ParentID),
		"parents":  parentTags,
		"selected": "tags",
	}))
}

func (i *IndexHandler) SaveTag(c *gin.Context) {
	var request vo.EditTagVo

	if err := c.Bind(&request); err != nil {
		c.Redirect(302, "/tags")
		return
	}
	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.Redirect(302, "/tags")
		return
	}
	showInHot := "Y"
	showInAll := "Y"
	openShowNum, err := strconv.Atoi(request.OpenShow)
	if err != nil {
		openShowNum = -1
	}
	var pid *uint
	if request.ShowInHot != "on" {
		showInHot = "N"
	}
	if request.ShowInAll != "on" {
		showInAll = "N"
	}
	if cast.ToInt(request.ParentID) > 0 {
		id := cast.ToUint(request.ParentID)
		pid = &id
	} else {
		pid = nil
	}
	log.Printf("request.ParentID is %+v", cast.ToInt(request.ParentID))
	if cast.ToInt(request.ID) == 0 {
		i.db.Save(&model.TbTag{
			Name:      request.Name,
			Desc:      request.Desc,
			ParentID:  pid,
			CssClass:  request.CssClass,
			ShowInHot: showInHot,
			ShowInAll: showInAll,
			OpenShow:  openShowNum,
		})
	} else {
		i.db.Model(&model.TbTag{}).Where("id = ?", request.ID).
			Updates(map[string]interface{}{
				"name":        request.Name,
				"desc":        request.Desc,
				"parent_id":   pid,
				"css_class":   request.CssClass,
				"show_in_hot": showInHot,
				"show_in_all": showInAll,
				"open_show":   openShowNum,
			})
	}

	c.Redirect(302, "/tags")
}

func (i *IndexHandler) AddTag(c *gin.Context) {
	var request vo.EditTagVo
	if err := c.Bind(&request); err != nil {
		c.JSON(403, nil)
		return
	}
	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.JSON(403, nil)
		return
	}
	var tag model.TbTag
	tag.Name = request.Name
	tag.Desc = request.Desc
	if request.ParentID != nil {
		tag.Parent = &model.TbTag{
			Model: gorm.Model{
				ID: *request.ParentID,
			},
		}
	}
	i.db.Create(&tag)
	c.JSON(200, nil)
}

func (i *IndexHandler) ToTags(c *gin.Context) {
	var tags []model.TbTag
	i.db.Model(&model.TbTag{}).Where("parent_id is null").Preload("Children", func(db *gorm.DB) *gorm.DB {
		return db.Order("name") // 对子标签进行排序
	}).Order("name").Find(&tags)
	c.HTML(200, "tags.html", OutputCommonSession(i.injector, c, gin.H{
		"tags":     tags,
		"selected": "tags",
	}))
}
func (i *IndexHandler) ToWaitApproved(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	var waitApprovedList []model.TbPost
	if userinfo != nil {
		if userinfo.Role == "admin" || userinfo.Role == "inspector" {
			i.db.Model(&model.TbPost{}).Preload("User").Preload("Tags").
				Where("status = 'Wait'").Order("created_at desc").
				Find(&waitApprovedList)
			if len(waitApprovedList) == 0 {
				c.Redirect(302, "/")
				return
			}
		} else {
			c.Redirect(302, "/")
			return
		}
	} else {
		c.Redirect(302, "/u/login")
		return
	}

	c.HTML(200, "wait.html", OutputCommonSession(i.injector, c, gin.H{
		"posts":        waitApprovedList,
		"waitApproved": len(waitApprovedList),
		"selected":     "approve",
	}))
}

func (i *IndexHandler) History(c *gin.Context) {
	userinfo := GetCurrentUser(c)

	page := c.DefaultQuery("p", "1")

	c.HTML(200, "index.html", OutputCommonSession(i.injector, c, gin.H{
		"selected": "history",
		"title":    "最新",
		"slogan":   "链接有趣内容，聚合真实想法，和真实的人一起筛内容，不靠算法也能刷到好东西。",
	}, QueryPosts(i.db, vo.QueryPostsRequest{
		Userinfo: userinfo,
		Page:     cast.ToInt64(page),
		Size:     25,
	})))
}

func (i *IndexHandler) ToComments(c *gin.Context) {
	page := c.DefaultQuery("p", "1")
	size := 25
	var comments []model.TbComment
	var total int64
	var totalPage int64
	pageNumber := cast.ToInt(page)
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "权限错误", "msg": "游客无法查看全部评论列表！",
		}))
		return
	}

	if userinfo != nil {
		subQuery := i.db.Table("tb_vote").Select("target_id").Where("tb_user_id = ? and type = 'COMMENT' and action ='UP'", userinfo.ID)

		i.db.Table("tb_comment c").Select("c.*,CASE WHEN vote.target_id IS NOT NULL THEN 1 ELSE 0  END AS up_voted").Joins("LEFT JOIN (?) AS vote ON c.id = vote.target_id", subQuery).Preload("Post").
			Preload("User").Order("created_at desc").Limit(int(size)).Offset((pageNumber - 1) * size).Find(&comments)
	} else {
		i.db.Model(model.TbComment{}).Preload("Post").
			Preload("User").Order("created_at desc").Limit(int(size)).Offset((pageNumber - 1) * size).Find(&comments)
	}

	i.db.Model(model.TbComment{}).Count(&total)
	totalPage = total / int64(size)
	if total%int64(size) > 0 {
		totalPage = totalPage + 1
	}
	c.HTML(200, "comments.html", OutputCommonSession(i.injector, c, gin.H{
		"selected":    "comment",
		"comments":    comments,
		"totalPage":   totalPage,
		"hasNext":     pageNumber < int(totalPage),
		"hasPrev":     pageNumber > 1,
		"currentPage": pageNumber,
	}))
}

func (i *IndexHandler) DelComment(c *gin.Context) {
	cid := c.Query("cid")
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}
	// 查出这条评论
	var item model.TbComment
	i.db.Model(&model.TbComment{}).Preload("Post").Where("cid = ?", cid).First(&item)
	// 如果是自己删除自己的评论，判断用户积分是否足够
	if userinfo.Role != "admin" && userinfo.ID == item.UserID {
		var user model.TbUser
		i.db.Where("ID = ?", userinfo.ID).First(&user)
		if user.Points-3 < 0 {
			c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
				"title": "删除失败", "msg": "积分不足，删除评论失败！",
			}))
			return
		}
	}
	// 判断这条评论是不是本人删除或者删除者是不是管理员
	if userinfo.Role != "admin" && userinfo.ID != item.UserID {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "权限错误", "msg": "非管理员只允许删除自己发布的评论！",
		}))
		return
	}
	// 发送提醒消息和删除评论
	var targetID uint
	var message model.TbMessage

	message.FromUserID = 999999999
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	message.Read = "N"

	targetID = item.ID
	if utf8.RuneCountInString(item.Content) > 10 {
		i := 0
		for j := range item.Content {
			if i == 10 {
				item.Content = item.Content[:j] + "..."
				break
			}
			i++
		}
	}
	message.ToUserID = item.UserID
	message.Content = fmt.Sprintf("你的评论被管理员删除 (<a class='bLink' href='/p/%s#c-%s'>%s</a>)",
		item.Post.Pid, item.CID, item.Content)
	var inspectLog model.TbInspectLog
	inspectLog.InspectType = "Comment"
	inspectLog.PostID = item.PostID
	inspectLog.Reason = "删除评论"
	inspectLog.Result = "deleted"
	inspectLog.Action = "deleted Comment"
	inspectLog.InspectorID = userinfo.ID
	inspectLog.Title = item.Content

	i.db.Transaction(func(tx *gorm.DB) error {
		// 删除评论
		if err := tx.Model(&model.TbComment{}).Where("id =?", targetID).Update("content", "**** 该评论已被删除 ****").Error; err != nil {
			return err
		}
		// 如果是管理员删除
		if userinfo.Role == "admin" && userinfo.ID != item.UserID {
			// 保存删除日志
			err := tx.Save(&inspectLog).Error
			if err != nil {
				return err
			}
			// 发送站内信提示
			if err := tx.Save(&message).Error; err != nil {
				return err
			}
		}
		// 扣除积分
		if item.UserID > 0 {
			handler := UserHandler{i.injector, i.db}
			err := handler.ChangePoints(item.UserID, 0, 3)
			if err != nil {
				return err
			}
		}
		return nil
	})

	refer := c.GetHeader("Referer")
	if refer == "" {
		refer = "/"
	}
	c.Redirect(302, refer)
}

func (i *IndexHandler) Vote(c *gin.Context) {
	id := c.Query("id")
	action := c.Query("action")
	targetType := c.Query("type")
	var vote model.TbVote
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}

	refer := c.GetHeader("Referer")
	if refer == "" {
		refer = "/"
	}

	uid := userinfo.ID
	itemUid := userinfo.ID

	var exists int64
	var targetID uint
	var message model.TbMessage

	message.FromUserID = 999999999
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	message.Read = "N"

	if targetType == "POST" {
		var item model.TbPost
		i.db.Model(&model.TbPost{}).Where("pid = ?", id).First(&item)
		targetID = item.ID
		itemUid = item.UserID
		if item.UserID == uid && action == "u" {
			c.Redirect(302, refer)
			return
		}
		message.ToUserID = item.UserID
		if action == "c" {
			message.Content = fmt.Sprintf("<a class='bLink' href='/u/profile/%d'>%s</a>收藏了你的主题<a class='bLink' href='/p/%s'>%s</a>",
				userinfo.ID, userinfo.Username, item.Pid, item.Title)
		} else if action == "u" {
			message.Content = fmt.Sprintf("<a class='bLink' href='/u/profile/%d'>%s</a>给你的主题<a class='bLink' href='/p/%s'>%s</a>点赞了",
				userinfo.ID, userinfo.Username, item.Pid, item.Title)
		}
	} else if targetType == "COMMENT" {
		var item model.TbComment
		i.db.Model(&model.TbComment{}).Preload("Post").Where("cid = ?", id).First(&item)
		targetID = item.ID
		itemUid = item.UserID
		if item.UserID == uid {
			log.Printf("comment item.UserID == uid ")

			c.Redirect(302, refer)
			return
		}
		message.ToUserID = item.UserID
		message.Content = fmt.Sprintf("<a class='bLink' href='/u/profile/%d'>%s</a>给你的<a class='bLink' href='/p/%s#c-%s'>评论</a>点赞了",
			userinfo.ID, userinfo.Username, item.Post.Pid, item.CID)
	}
	var col string
	if action == "u" {
		vote.Action = "UP"
		col = "upVote"
	} else {
		vote.Action = "Collect"
		col = "collectVote"
	}
	vote.UserID = uid
	vote.TargetID = targetID
	vote.Type = targetType
	// 取消收藏的逻辑
	if action == "cd" {
		i.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.TbVote{}).Where("target_id = ? and tb_user_id = ?  and type = ? and action = ?", targetID, uid, targetType, vote.Action).Unscoped().Delete(&model.TbVote{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.TbPost{}).Where("id =?", targetID).Update(col, gorm.Expr(fmt.Sprintf("\"%s\"", col)+"-1")).Error; err != nil {
				return err
			}
			return nil
		})
		c.Redirect(302, refer)
		return
	}
	if i.db.Model(&model.TbVote{}).Where("target_id = ? and tb_user_id = ?  and type = ? and action = ?", targetID, uid, targetType, vote.Action).Count(&exists); exists == 0 {
		i.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&vote).Error; err != nil {
				return err
			}
			if targetType == "POST" {
				if err := tx.Model(&model.TbPost{}).Where("id =?", targetID).Update(col, gorm.Expr(fmt.Sprintf("\"%s\"", col)+"+1")).Error; err != nil {
					return err
				}
			} else if targetType == "COMMENT" {
				if err := tx.Model(&model.TbComment{}).Where("id =?", targetID).Update(col, gorm.Expr(fmt.Sprintf("\"%s\"", col)+"+1")).Error; err != nil {
					return err
				}
			}
			// 如果是自己操作自己发布的内容不发送消息和增加积分
			if itemUid != uid {
				if err := tx.Save(&message).Error; err != nil {
					return err
				}
				if message.ToUserID > 0 {
					handler := UserHandler{i.injector, i.db}
					err := handler.ChangePoints(message.ToUserID, 1, 1)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
	}

	c.Redirect(302, refer)
}

func (i *IndexHandler) Moderation(c *gin.Context) {
	page := c.DefaultQuery("p", "1")
	size := 25
	var logs []model.TbInspectLog
	var total int64
	var totalPage int64
	pageNumber := cast.ToInt(page)

	i.db.Model(&model.TbInspectLog{}).Preload("Inspector").Preload("Post").Limit(size).Offset((pageNumber - 1) * size).Find(&logs)
	i.db.Model(&model.TbInspectLog{}).Count(&total)

	totalPage = total / int64(size)
	if total%int64(size) > 0 {
		totalPage = totalPage + 1
	}
	c.HTML(200, "moderation.html", OutputCommonSession(i.injector, c, gin.H{
		"logs":        logs,
		"totalPage":   totalPage,
		"hasNext":     pageNumber < int(totalPage),
		"hasPrev":     pageNumber > 1,
		"currentPage": pageNumber,
	}))
}

func (i *IndexHandler) SearchByDomain(c *gin.Context) {
	userinfo := GetCurrentUser(c)

	domainName := c.Param("domainName")

	page := c.DefaultQuery("p", "1")

	c.HTML(200, "index.html", OutputCommonSession(i.injector, c, gin.H{
		"selected": "history",
		"title":    domainName + " 相关热议汇总",
		"slogan":   "链接有趣内容，聚合真实想法，和真实的人一起筛内容，不靠算法也能刷到好东西。",
	}, QueryPosts(i.db, vo.QueryPostsRequest{
		Userinfo:  userinfo,
		Domain:    domainName,
		OrderType: "",
		Page:      cast.ToInt64(page),
		Size:      25,
	})))
}

func (i *IndexHandler) ToSettings(c *gin.Context) {

	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.Redirect(302, "/")
		return
	}

	var settings model.TbSettings
	var saveSettingsRequest vo.SaveSettingsRequest
	if errors.Is(i.db.First(&settings).Error, gorm.ErrRecordNotFound) {
		saveSettingsRequest.RegMode = "hotnews"
		c.HTML(200, "settings.html", OutputCommonSession(i.injector, c, gin.H{
			"selected": "settings",
		}))
		return
	}

	c.HTML(200, "settings.html", OutputCommonSession(i.injector, c, gin.H{
		"selected": "settings",
	}))
}

func (i *IndexHandler) SaveSettings(c *gin.Context) {
	var request vo.SaveSettingsRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(403, nil)
		return
	}
	var settings model.TbSettings
	i.db.First(&settings)

	settings.Content = model.SaveSettingsRequest(request)
	i.db.Save(&settings)

	c.Redirect(302, "/settings")

}

func (i *IndexHandler) RemoveTag(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil || userinfo.Role != "admin" {
		c.Redirect(302, "/tags")
		return
	}
	tagId, _ := strconv.Atoi(c.Param("tagId"))
	var tag model.TbTag
	i.db.Preload("Posts").First(&tag, "id = ?", tagId)
	i.db.Delete(&tag.Posts)
	i.db.Delete(&tag)
	c.Redirect(302, "/tags")
}

// Activate 发送激活邮件
func (i *IndexHandler) Activate(c *gin.Context) {
	uid := c.Query("id")
	//key := c.Query("key")
	userinfo := GetCurrentUser(c)
	if uid == "" || userinfo == nil || uid != strconv.FormatUint(uint64(userinfo.ID), 10) {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "Error", "msg": "参数错误！",
		}))
		return
	}
	data := []byte(userinfo.Email)
	hash := md5.Sum(data)
	key := hex.EncodeToString(hash[:])
	siteName := os.Getenv("SiteName")
	siteUrl := os.Getenv("SiteUrl")
	// 将激活邮件发送给用户
	content := "您好，<br><br>收到此邮件是因为您在<b>竹林</b>网站上进行了注册，<br><br>请点击链接激活账号：" + siteUrl + "/u/status?id=" + uid + "&key=" + key
	msg := utils.Email{}.Send(userinfo.Email, "["+siteName+"] 账户激活邮件", content)
	if msg != "Success" {
		c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
			"title": "系统异常", "msg": "激活邮件发送异常，请稍后重试！",
		}))
		return
	}
	c.HTML(200, "result.html", OutputCommonSession(i.injector, c, gin.H{
		"title": "Success", "msg": "激活邮件已发送，请查收邮箱！",
	}))
}

// UploadImg 头像上传
func (i *IndexHandler) UploadImg(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.JSON(400, gin.H{"error": "未登录用户禁止上传图片！"})
		return
	}
	// 设置最大上传文件大小为200KB
	maxMemory := int64(100 * 1024) // 200KB转换为字节，并且转换为int64类型
	// 解析表单数据，设置最大内存使用量为maxMemory
	err := c.Request.ParseMultipartForm(maxMemory)
	if err != nil {
		c.JSON(200, gin.H{"code": 400, "file_path": "", "message": "图片太大，请压缩至 100KB 以内再次上传！"})
		return
	}
	// 从表单中获取文件
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(200, gin.H{"code": 400, "file_path": "", "message": err.Error()})
		return
	}
	// 判断上传的是不是图片
	mimeType := file.Header.Get("Content-Type")
	fmt.Println(mimeType)
	if mimeType[:6] != "image/" {
		c.JSON(200, gin.H{"code": 400, "file_path": "", "message": "搞事情？只允许上传图片！"})
		return
	}
	// 检查文件大小是否超过限制
	if file.Size > maxMemory {
		c.JSON(200, gin.H{"code": 400, "file_path": "", "message": "图片太大，请压缩至 100KB 以内再次上传！"})
		return
	}
	// 指定文件应保存的路径
	filePath := fmt.Sprintf("./static/user_avatar/%d.jpg", userinfo.ID)
	//filePath := fmt.Sprintf("./static/user_avatar/%d.jpg", 1)
	// 检查文件是否存在，如果存在先删除
	if _, err = os.Stat(filePath); err == nil {
		// 文件存在，删除它
		err = os.Remove(filePath)
		if err != nil {
			// 处理删除文件时可能出现的错误
			c.JSON(200, gin.H{"code": 400, "file_path": "", "message": "Failed to delete the existing file"})
			return
		}
	}
	// 将文件保存到服务器上
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(200, gin.H{"code": 400, "file_path": "", "message": "Failed to save the file"})
		return
	}
	filePath = strings.Replace(filePath, "./", "/", -1)
	// 返回成功响应
	c.JSON(200, gin.H{"code": 200, "message": "File uploaded successfully", "file_path": filePath})
}
