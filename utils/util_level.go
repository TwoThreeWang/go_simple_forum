/**
 * @Author: wangcheng
 * @Author: job_wangcheng@163.com
 * @Date: 2024/7/22 下午2:56
 * @Description: 根据积分计算用户等级
 */

package utils

import (
	"strconv"
)

// GetUserLevel "根据用户积分判断用户等级，新注册用户默认等级0，每增加100等级增加1"
// "<100时是等级零"
// "100-200是等级一"
// "每增加100增加一个等级"
//
//	func GetUserLevel(score int) string {
//		// 使用 math.Floor 向下取整
//		return strconv.Itoa(int(math.Floor(float64(score) / 100)))
//	}
func GetUserLevel(points int) string {
	level := 0
	threshold := 50 // 每级的积分门槛

	for points >= threshold {
		level++
		threshold += 50 * level // 每升一级后，门槛递增
	}
	return strconv.Itoa(level)
}
