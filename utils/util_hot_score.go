package utils

import (
	"go_simple_forum/model"
	"math"
	"time"

	"gorm.io/gorm"
)

// CalculateHotScore 计算帖子热门分数
func CalculateHotScore(db *gorm.DB, pid string) {
	// 新帖初期会有较高热度；
	// 随时间自然下滑；
	// 如果用户有评论或作者更新内容，热度会重新上升；
	// 浏览刷量不会轻易改变排名，因为系数较小。
	// 收藏和评论真正推动榜单。
	// 这套模型的核心是：热度 = 行为得分 × 活跃度加成 ÷ 时间衰减

	// 根据ID查询帖子
	var post model.TbPost
	db.Model(&model.TbPost{}).Where("pid= ?", pid).First(&post)
	// var commentCount int64
	// db.Model(&model.TbComment{}).Where("post_id = ? and user_id != ?", post.ID, post.UserID).Count(&commentCount)
	// 行为得分基础值：帖子的原始“势能”，越多人点赞/评论/收藏/浏览，势能越高
	base := 0.1*float64(post.ClickVote) + // 点击量用户投入最轻，权重0.1
		1.0*float64(post.UpVote) + // 点赞用户投入轻，认可，权重1.0
		1.5*float64(post.CommentCount) + // 评论用户投入中，参与度高，权重1.5
		2.0*float64(post.CollectVote) // 收藏说明有长期价值，权重2.0

	hoursSinceCreate := time.Since(post.CreatedAt).Hours()
	hoursSinceUpdate := time.Since(post.UpdatedAt).Hours()
	// 时间衰减，帖子越老，热度越低
	ageDecay := math.Pow(hoursSinceCreate+0.5, 1.3)
	// 最近更新，活跃加成，最近有人互动的帖子，会让热度恢复或保持高位，老帖在有新互动（比如更新）后能恢复部分热度
	updateBoost := 1 / (1 + hoursSinceUpdate/24)

	hotScore := base * updateBoost / ageDecay
	db.Model(&model.TbPost{}).Where("id= ?", post.ID).UpdateColumn("point", hotScore)
}
