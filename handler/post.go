package handler

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"go_simple_forum/model"
	"go_simple_forum/utils"
	"go_simple_forum/vo"

	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type PostHandler struct {
	injector *do.Injector
	db       *gorm.DB
}

func NewPostHandler(injector *do.Injector) (*PostHandler, error) {
	return &PostHandler{
		injector: injector,
		db:       do.MustInvoke[*gorm.DB](injector),
	}, nil
}

// ClickPost 帖子点击量增加
func (p PostHandler) ClickPost(c *gin.Context) {
	pid := cast.ToString(c.Param("pid"))
	if pid == "" {
		c.JSON(400, gin.H{"error": "参数不能为空"})
		return
	}
	if err := p.db.Model(&model.TbPost{}).Where("pid =?", pid).Update("clickVote", gorm.Expr(fmt.Sprintf("\"%s\"", "clickVote")+"+1")).Error; err != nil {
		fmt.Println("帖子点击量增加报错：" + err.Error())
	}
	// 刷新帖子热门分数
	go utils.CalculateHotScore(p.db, pid)
	c.JSON(200, gin.H{"message": "success"})
}

func (p PostHandler) DoUpdate(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	pid := cast.ToString(c.Param("pid"))
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}
	if userinfo.Role != "admin" {
		if userinfo == nil {
			c.Redirect(302, "/")
			return
		}
	}
	var request vo.NewPostRequest
	if err := c.Bind(&request); err != nil {
		c.Redirect(302, "/")
		return
	}
	// 验证 Turnstile 令牌
	if request.CfTurnstile != "" {
		remoteIP := c.ClientIP()
		_, err := utils.VerifyTurnstileToken(c, request.CfTurnstile, remoteIP)
		if err != nil {
			c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
				"title": "参数错误", "msg": "验证 Turnstile 令牌失败：" + err.Error(),
			}))
			return
		}
	} else {
		c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
			"title": "参数错误", "msg": "验证 Turnstile 令牌失败：缺少验证参数",
		}))
		return
	}
	var post model.TbPost
	if err := p.db.Preload("Tags").First(&post, "pid = ?", pid).Error; err != nil {
		c.Redirect(302, "/")
		return
	}
	var tags []model.TbTag
	p.db.Model(&model.TbTag{}).Find(&tags, "id in ?", request.TagIDs)

	tx := p.db.Model(&post)
	tx.Association("Tags").Unscoped().Clear()

	post.Title = request.Title
	post.Content = request.Content
	post.Link = request.Link
	post.Type = request.Type
	host := ""
	post.Tags = tags
	if request.Type == "link" {
		urlParsed, _ := url.Parse(request.Link)
		host = urlParsed.Host
		if strings.HasPrefix(host, "www.") {
			_, host, _ = strings.Cut(host, "www.")
		}
	} else {
		post.Link = ""
	}
	post.Domain = host
	post.UpdatedAt = time.Now()
	if userinfo.Role == "admin" {
		if request.Top == "on" {
			post.Top = 1
		} else {
			post.Top = 0
		}
	}
	p.db.Save(&post)
	// 刷新帖子热门分数
	go utils.CalculateHotScore(p.db, pid)
	c.Redirect(302, "/p/"+post.Pid)
}

func (p PostHandler) ToEdit(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	pid := cast.ToString(c.Param("pid"))
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}
	if userinfo.Role != "admin" {
		if userinfo == nil {
			c.Redirect(302, "/")
			return
		}
	}
	var post model.TbPost
	if err := p.db.Preload("Tags").First(&post, "pid = ?", pid).Error; err != nil {
		if userinfo == nil {
			c.Redirect(302, "/")
			return
		}
	}
	var tempTags []model.TbTag
	p.db.Model(&model.TbTag{}).Preload("Parent").Where("parent_id is null").Preload("Children").Find(&tempTags)
	c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
		"post":     post,
		"selected": "new",
		"tags":     tempTags,
	}))
}
func (p PostHandler) Detail(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	// 查询帖子内容
	var posts []model.TbPost
	result := QueryPosts(p.db, vo.QueryPostsRequest{
		Userinfo:  userinfo,
		Page:      1,
		Size:      25,
		PostPID:   cast.ToString(c.Param("pid")),
		OrderType: "single",
	})

	posts = result["posts"].([]model.TbPost)
	// 没有查询到结果
	if len(posts) == 0 {
		c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
			"title": "Nothing To Show", "msg": "没有内容可以展示！",
		}))
		return
	}
	// 验证用户权限是否符合查看权限
	role := -1
	if userinfo != nil {
		roleStr := userinfo.Role
		if roleStr == "admin" {
			role = 99999
		} else {
			num, err := strconv.Atoi(roleStr)
			if err != nil {
				role = 0
			} else {
				role = num
			}
		}
	}
	for _, post := range posts {
		for _, tag := range post.Tags {
			if tag.OpenShow > role {
				c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
					"title": "权限错误", "msg": fmt.Sprintf("等级 LV.%d 以上才可以查看该标签下的内容！", tag.OpenShow),
				}))
				return
			}
		}
	}

	// 点赞及评论数据
	var uid uint = 0
	var rootComments []model.TbComment
	if userinfo != nil {
		uid = userinfo.ID
		subQuery := p.db.Table("tb_vote").Select("target_id").Where("tb_user_id = ? and type = 'COMMENT' and action ='UP'", uid)

		p.db.Table("tb_comment c").Select("c.*,CASE WHEN vote.target_id IS NOT NULL THEN 1 ELSE 0  END AS up_voted").Joins("LEFT JOIN (?) AS vote ON c.id = vote.target_id", subQuery).
			Preload("User").Where("post_id = ? and parent_comment_id is null", posts[0].ID).Order("created_at desc").Find(&rootComments)

	} else {
		p.db.Table("tb_comment c").Select("c.*").
			Preload("User").Where("post_id = ? and parent_comment_id is null", posts[0].ID).Order("created_at desc").
			Find(&rootComments)

	}

	buildCommentTree(&rootComments, p.db, uid)
	posts[0].Comments = rootComments

	// 获取相关文章推荐
	var relatedPosts []model.TbPost
	if len(posts) > 0 {
		// 获取当前文章的标签IDs
		var tagIDs []uint
		for _, tag := range posts[0].Tags {
			tagIDs = append(tagIDs, tag.ID)
		}

		// 查询具有相同标签的其他文章
		if len(tagIDs) > 0 {
			p.db.Preload("Tags").Preload("User").Where("id != ?", posts[0].ID).
				Joins("JOIN tb_post_tag ON tb_post_tag.tb_post_id = tb_post.id").
				Where("tb_post_tag.tb_tag_id IN ?", tagIDs).
				Group("tb_post.id").
				Order("COUNT(DISTINCT tb_post_tag.tb_tag_id) DESC, tb_post.point DESC").
				Limit(5).Find(&relatedPosts)
		}
	}

	c.HTML(200, "post.html", OutputCommonSession(p.injector, c, gin.H{
		"posts":        posts,
		"relatedPosts": relatedPosts,
		"selected":     "detail",
	}))
}

func buildCommentTree(comments *[]model.TbComment, db *gorm.DB, uid uint) {
	subQuery := db.Table("tb_vote").Select("target_id").Where("tb_user_id = ? and type = 'COMMENT' and action ='UP'", uid)
	for i := range *comments {
		var children []model.TbComment
		if uid > 0 {
			db.Table("tb_comment c").Select("c.*,CASE WHEN vote.target_id IS NOT NULL THEN 1 ELSE 0  END AS up_voted").
				Joins("LEFT JOIN (?) AS vote ON c.id = vote.target_id", subQuery).Preload("User").Where("post_id = ? and parent_comment_id = ?", (*comments)[i].PostID, (*comments)[i].ID).Find(&children)
		} else {
			db.Model(&model.TbComment{}).
				Joins("LEFT JOIN (?) AS vote ON c.id = vote.target_id", subQuery).Preload("User").Where("post_id = ? and parent_comment_id = ?", (*comments)[i].PostID, (*comments)[i].ID).Find(&children)
		}
		(*comments)[i].Comments = children
		buildCommentTree(&(*comments)[i].Comments, db, uid)
	}
}

func (p PostHandler) Add(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}

	uid := userinfo.ID
	// 判断用户是否激活
	var user model.TbUser
	p.db.Model(model.TbUser{}).Where("id=?", uid).First(&user)
	status := "Active"
	if user.Status != "Active" {
		msg := "禁止用户不允许发布"
		if user.Status == "Wait" {
			msg = "未激活用户不允许发布，请到个人中心邮箱激活账户！"
		}
		c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
			"title": "Error",
			"msg":   msg,
		}))
		return
	}
	// 使用当前时间作为随机数种子
	rand.Seed(time.Now().UnixNano())
	// 生成 1-10 之间的随机整数
	points := rand.Intn(5) + 1 // rand.Intn(10) 生成 0-9 的随机数，+1 使其变为 1-10
	// 等级 2 以下的用户发帖需要审核
	if userinfo.Role != "admin" {
		role, err := strconv.Atoi(userinfo.Role)
		if role < 2 || err != nil {
			status = "Wait"
			points = 0
		}
	}
	var tempTags []model.TbTag
	p.db.Model(&model.TbTag{}).Preload("Parent").Where("parent_id is null").Preload("Children").Find(&tempTags)

	var request vo.NewPostRequest
	if err := c.Bind(&request); err != nil {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "参数异常",
			"selected": "new",
			"tags":     tempTags,
		}))
		return
	}
	// 验证 Turnstile 令牌
	if request.CfTurnstile != "" {
		remoteIP := c.ClientIP()
		_, err := utils.VerifyTurnstileToken(c, request.CfTurnstile, remoteIP)
		if err != nil {
			c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
				"msg":      "验证 Turnstile 令牌失败：" + err.Error(),
				"selected": "new",
				"tags":     tempTags,
			}))
			return
		}
	} else {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "验证 Turnstile 令牌失败：缺少验证参数",
			"selected": "new",
			"tags":     tempTags,
		}))
		return
	}
	if len(request.Link) > 1024 {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "网址链接太长了，最大长度1024",
			"selected": "new",
			"tags":     tempTags,
		}))
		return
	}
	log.Printf("params:%+v", request)
	if len(request.TagIDs) == 0 || len(request.TagIDs) > 5 {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "标签最少1个,最多5个",
			"selected": "new",
			"tags":     tempTags,
		}))
		return
	}
	if request.Type == "" {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "类型必填",
			"selected": "new",
			"tags":     tempTags,
		}))
		return
	}
	if request.Type == "link" && request.Link == "" {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "分享类的链接是必填项",
			"selected": "new",
			"tags":     tempTags,
		}))
		return
	}
	var tags []model.TbTag
	for _, v := range request.TagIDs {
		tags = append(tags, model.TbTag{
			Model: gorm.Model{ID: v},
		})
	}

	host := ""
	if request.Type == "link" {
		urlParsed, _ := url.Parse(request.Link)
		host = urlParsed.Host
		if strings.HasPrefix(host, "www.") {
			_, host, _ = strings.Cut(host, "www.")
		}
	}

	top := 0
	if userinfo.Role == "admin" && request.Top == "on" {
		top = 1
	}

	post := model.TbPost{
		Title:        strings.Trim(request.Title, " "),
		Link:         strings.Trim(request.Link, " "),
		Status:       status,
		Content:      strings.Trim(request.Content, " "),
		UpVote:       0,
		CollectVote:  0,
		Type:         request.Type,
		Tags:         tags,
		User:         model.TbUser{Model: gorm.Model{ID: uid}},
		Domain:       host,
		Pid:          RandStringBytesMaskImpr(8),
		CommentCount: 0,
		Top:          top,
	}

	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&post).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.TbUser{Model: gorm.Model{ID: uid}}).Update("postCount", user.PostCount+1).Update("points", gorm.Expr("\"points\" + ?", points)).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.HTML(200, "new.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":      "系统错误",
			"selected": "new",
		}))
		return
	}
	// 提交到google index api
	SiteUrl := os.Getenv("SiteUrl")
	herfs := []string{
		SiteUrl + "/p/" + post.Pid,
	}
	go utils.Submit2Google(herfs)
	if status == "Active" {
		c.Redirect(302, "/p/"+post.Pid)
	} else if status == "Wait" {
		c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
			"msg":   "发布成功，等待管理员审核后展示！",
			"title": "Success",
		}))
	} else {
		c.Redirect(302, "/")
	}
	return
}

func (p PostHandler) AddComment(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	if userinfo == nil {
		c.Redirect(302, "/u/login")
		return
	}
	uid := userinfo.ID
	// 判断用户激活状态
	var user model.TbUser
	p.db.Model(model.TbUser{}).Where("id=?", uid).First(&user)
	status := user.Status
	if status != "Active" {
		msg := "禁止用户不允许评论"
		if status == "Wait" {
			msg = "未激活用户不允许评论，请到个人中心邮箱激活账户！"
		}
		c.HTML(200, "result.html", OutputCommonSession(p.injector, c, gin.H{
			"title": "Error",
			"msg":   msg,
		}))
		return
	}
	var comment model.TbComment
	var request vo.NewCommentRequest
	err := c.Bind(&request)
	if err != nil {
		c.Redirect(302, "/")
		return
	}
	comment.PostID = request.PostID

	var message model.TbMessage
	var post model.TbPost
	p.db.First(&post, "id = ?", request.PostID)

	comment.Content = request.Content
	comment.UserID = uid
	comment.UpVote = 0
	comment.DownVote = 0
	comment.CID = RandStringBytesMaskImpr(8)

	if request.ParentCommentId == 0 {
		comment.ParentCommentID = nil
		if userinfo.ID != post.UserID {
			message.ToUserID = post.UserID
			message.Content = fmt.Sprintf("<a class='bLink' href='/u/profile/%d'>%s</a>在<a class='bLink' href='/p/%s'>[%s]</a>中回复了你的主题",
				userinfo.ID, userinfo.Username, post.Pid+"#c-"+comment.CID, post.Title)
		}
	} else {
		var parent model.TbComment
		p.db.First(&parent, "id = ?", request.ParentCommentId)
		comment.ParentCommentID = &request.ParentCommentId
		if userinfo.ID != parent.UserID {
			message.ToUserID = parent.UserID
			message.Content = fmt.Sprintf("<a class='bLink' href='/u/profile/%d'>%s</a>在<a class='bLink' href='/p/%s'>[%s]</a>回复了<a class='bLink' href='/p/%s'>你的评论</a>",
				userinfo.ID, userinfo.Username, post.Pid, post.Title, post.Pid+"#c-"+parent.CID)
		}

	}
	message.FromUserID = 999999999

	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	message.Read = "N"

	var redirectUrl = "/p/" + request.PostPID + "#c-" + comment.CID

	err = p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.TbPost{}).Where("id = ?", request.PostID).Update("commentCount", gorm.Expr("\"commentCount\" + 1")).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.TbUser{}).Where("id = ?", userinfo.ID).Update("commentCount", gorm.Expr("\"commentCount\" + 1")).Update("points", gorm.Expr("\"points\" + 1")).Error; err != nil {
			return err
		}
		if message.Content != "" {
			if err := tx.Save(&message).Error; err != nil {
				return err
			}
		}
		// 回复评论，被回复方加积分
		if message.ToUserID > 0 && message.ToUserID != comment.UserID {
			handler := UserHandler{p.injector, p.db}
			err := handler.ChangePoints(message.ToUserID, 1, 1)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.Redirect(302, "/")
		return
	}
	// 刷新帖子热门分数
	go utils.CalculateHotScore(p.db, post.Pid)
	c.Redirect(302, redirectUrl)
}

func (p PostHandler) SearchByTag(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	page := c.DefaultQuery("p", "1")

	tagName := strings.Split(c.Param("tag"), ",")

	c.HTML(200, "index.html", OutputCommonSession(p.injector, c, gin.H{
		"selected": "history",
	}, QueryPosts(p.db, vo.QueryPostsRequest{
		Userinfo:  userinfo,
		Tags:      tagName,
		OrderType: "",
		Page:      cast.ToInt64(page),
		Size:      25,
	})))
}

func (p PostHandler) SearchByParentTag(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	page := c.DefaultQuery("p", "1")

	var tags []string
	p.db.Table("tb_tag").
		Select("name").
		Where("parent_id = (select id from tb_tag a where a.name = ?)", c.Param("tag")).Scan(&tags)

	c.HTML(200, "index.html", OutputCommonSession(p.injector, c, gin.H{
		"selected": "history",
	}, QueryPosts(p.db, vo.QueryPostsRequest{
		Userinfo:  userinfo,
		Tags:      tags,
		OrderType: "",
		Page:      cast.ToInt64(page),
		Size:      25,
	})))
}

func (p PostHandler) SearchByType(c *gin.Context) {
	userinfo := GetCurrentUser(c)
	page := c.DefaultQuery("p", "1")
	typeName := c.Param("type")
	c.HTML(200, "index.html", OutputCommonSession(p.injector, c, gin.H{
		"selected": "history",
	}, QueryPosts(p.db, vo.QueryPostsRequest{
		Userinfo:  userinfo,
		Type:      typeName,
		OrderType: "",
		Page:      cast.ToInt64(page),
		Size:      25,
	})))
}

func QueryPosts(db *gorm.DB, request vo.QueryPostsRequest) gin.H {

	tx := db.Table("tb_post p").Distinct().Where("status = 'Active'")
	if request.Type != "" {
		tx.Where("type = ?", request.Type)
	}
	if request.Begin != nil {
		tx.Where("p.created_at >= ?", request.Begin)
	}
	if request.End != nil {
		tx.Where("p.created_at <= ?", request.End)
	}
	if request.Domain != "" {
		tx.Where("domain = ?", request.Domain)
	}
	if request.PostPID != "" {
		tx.Where("pid = ?", request.PostPID)
	}
	if request.Q != "" {
		tx.Where("title like ? or content like ?", "%"+request.Q+"%", "%"+request.Q+"%")
	}
	if request.Userinfo != nil {
		subQuery := db.Table("tb_vote").Select("target_id").Where("tb_user_id = ? and type = 'POST' and action ='UP'", request.Userinfo.ID)
		tx.Joins("LEFT JOIN (?) AS vote ON p.id = vote.target_id", subQuery)
		subQueryCollect := db.Table("tb_vote").Select("target_id").Where("tb_user_id = ? and type = 'POST' and action ='Collect'", request.Userinfo.ID)
		tx.Joins("LEFT JOIN (?) AS vote_collect ON p.id = vote_collect.target_id", subQueryCollect)

	}
	tx.InnerJoins(",tb_post_tag ptw,tb_tag tw")
	tx.Where("tw.id = ptw.tb_tag_id and ptw.tb_post_id = p.id")
	if len(request.Tags) > 0 {
		tx.InnerJoins(",tb_post_tag pt,tb_tag t")
		tx.Where("t.id = pt.tb_tag_id and pt.tb_post_id = p.id")
		tx.Where("t.name in (?)", request.Tags)
		tx.Order("p.top desc,p.created_at desc")
	} else if request.OrderType == "index" {
		tx.Where("not exists (select 1 from tb_post_tag pt,tb_tag t where t.id = pt.tb_tag_id and pt.tb_post_id = p.id and t.show_in_hot = 'N')")
		//tx.Where("p.created_at >= current_date - interval '7 day' and p.point >= 0")
		tx.Order("p.top desc ,p.point desc,p.created_at desc")
	} else if request.OrderType == "rss" {
		tx.Where("not exists (select 1 from tb_post_tag pt,tb_tag t where t.id = pt.tb_tag_id and pt.tb_post_id = p.id and t.show_in_all = 'N')")
		tx.Order("p.created_at desc")
	} else if request.OrderType == "" {
		tx.Where("not exists (select 1 from tb_post_tag pt,tb_tag t where t.id = pt.tb_tag_id and pt.tb_post_id = p.id and t.show_in_all = 'N')")
		tx.Order("p.top desc,p.created_at desc")
	}

	var total int64
	tx.Distinct("p.id").Count(&total)

	var posts []model.TbPost

	if request.Userinfo != nil {
		tx.Select("p.*,CASE WHEN vote.target_id IS NOT NULL THEN 1 ELSE 0  END AS up_voted,CASE WHEN vote_collect.target_id IS NOT NULL THEN 1 ELSE 0  END AS collect_voted")
	} else {
		tx.Select("p.*")
	}
	tx.Preload("Tags").Preload("User").
		Limit(int(request.Size)).
		Offset(int((request.Page - 1) * request.Size)).
		Find(&posts)

	totalPage := total / request.Size

	if total%request.Size > 0 {
		totalPage = totalPage + 1
	}
	return gin.H{
		"posts":       posts,
		"totalPage":   totalPage,
		"hasNext":     request.Page < totalPage,
		"hasPrev":     request.Page > 1,
		"currentPage": cast.ToInt(request.Page),
	}
}
