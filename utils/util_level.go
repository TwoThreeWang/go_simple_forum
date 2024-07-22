/**
 * @Author: wangcheng
 * @Author: job_wangcheng@163.com
 * @Date: 2024/7/22 下午2:56
 * @Description: 根据积分计算用户等级
 */

package utils

import (
	"math"
	"strconv"
)

// GetUserLevel "根据用户积分判断用户等级，新注册用户默认等级0，每增加50等级增加1"
// "<10时是等级零"
// "10-50是等级一"
// "50-100是等级二
// "之后每增加50增加一个等级"
func GetUserLevel(score int) string {
	// 使用 math.Floor 向下取整
	return strconv.Itoa(int(math.Floor(float64(score) / 50)))
}
