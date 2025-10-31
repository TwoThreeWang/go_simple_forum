package task

import (
	"go_simple_forum/model"
	"go_simple_forum/utils"
	"log"
	"math"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/samber/do"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func StartPostTask(i *do.Injector) {
	s := gocron.NewScheduler()
	s.Every(1).Hours().Do(func(i *do.Injector) {
		log.Printf("start refresh last 15 days post points.")
		db := do.MustInvoke[*gorm.DB](i)

		var postIDs []string
		db.Model(&model.TbPost{}).Select("pid").Where("created_at >= now() - interval '15 day' and status = 'Active'").Pluck("id", &postIDs)
		for _, pid := range postIDs {
			utils.CalculateHotScore(db, pid)
		}
		log.Printf("end of refresh last 15 days post points.")
	}, i)
	s.Start()
}

func StartPostTaskBak(i *do.Injector) {
	s := gocron.NewScheduler()
	s.Every(1).Hours().Do(func(i *do.Injector) {
		log.Printf("start refresh last 15 days post points.")
		db := do.MustInvoke[*gorm.DB](i)

		var posts []model.TbPost
		db.Model(&model.TbPost{}).Where("created_at >= now() - interval '15 day' and status = 'Active'").Scan(&posts)
		g := 1.80
		for _, post := range posts {
			var commentCount int64
			db.Model(&model.TbComment{}).Where("post_id = ? and user_id != ?", post.ID, post.UserID).Count(&commentCount)

			p := float64(post.UpVote)*1.5 + cast.ToFloat64(commentCount)*1.2 + float64(post.CollectVote)*1.2 + float64(post.ClickVote)*1
			t := time.Now().Sub(post.CreatedAt).Hours()

			point := p / math.Pow(t+2, g)

			db.Model(&model.TbPost{}).Where("id= ?", post.ID).Update("point", point)
		}
		log.Printf("end of refresh last 15 days post points.")
	}, i)
	s.Start()
}
