# Go开发的极简论坛

## 项目简介

一个基于Go语言开发的轻量级论坛系统，具有以下特点：

- 极简设计：得益于Go语言强大的内嵌静态资源功能，镜像包仅6.29MB，运行时内存占用仅28MB
- 功能完善：支持Markdown编辑器、用户管理、积分系统、标签管理等功能
- 易于部署：支持Docker容器化部署，配置简单
- SEO友好：支持sitemap、RSS订阅

本项目基于 [https://github.com/kingwrcy/hotnews](https://github.com/kingwrcy/hotnews) 修改开发，感谢原作者的贡献。

在源代码的基础上增改的功能：

- [x] 发布及评论的编辑器改为 md 编辑器，文章内容保存为 md 格式
- [x] 优化文章内容行间距
- [x] 评论及文章内容链接改为新标签页打开
- [x] 头像可以设置自定义头像
- [x] 用户信息修改功能
- [x] 邮件重置密码功能
- [x] 增加404页面
- [x] 标签增加对游客隐藏选项
- [x] 新注册用户邮箱激活功能
- [x] 管理员管理用户状态（Wait：等待激活；Active：活跃用户；Banned：禁止用户）
- [x] 管理员管理文章状态（Wait：等待审核；Active：正常；Rejected：删除）
- [x] 增加邮箱登录支持
- [x] 增加sitemap
- [x] 管理员删除评论功能
- [x] 增加积分功能
- [x] 增加签到功能
- [x] 积分兑换邀请码
- [x] 用户主页改用ID，不用用户名
- [x] 热门帖子计算由7天内帖子改为15天内帖子，定时任务改为60分钟刷新一次排名
- [x] 信任级别小于2的用户发帖需要审核
- [x] 帖子审核通过或删除发站内通知
- [x] 默认头像改为使用本地图片
- [x] 自定义头像本地上传功能
- [x] SEO优化
- [x] 评论增加emoji
- [x] 增加帖子收藏功能
- [x] 增加 Google 一键登录功能
- [x] 优化黑暗模式，增加自动模式切换
- [x] 邮件发送改为使用 smtp
- [x] 增加 RSS
- [x] 增加图片代理接口，优化 google 头像显示
- [x] 标签增加阅读等级设置

### Demo

竹林：[https://zhulink.vip](https://zhulink.vip)

![论坛截图](https://openai-75050.gzc.vod.tencent-cloud.com/openaiassets_5ba4ebcbd2030fee5ac43c38e41a0f41_2579861720144999302.png)

## 主要功能

### 内容管理
- Markdown编辑器支持，文章内容以MD格式保存
- 文章状态管理（等待审核/正常/删除）
- 评论管理与emoji表情支持
- 帖子收藏功能
- 文章内容链接新标签页打开

### 用户系统
- 邮箱注册与登录
- Google一键登录集成
- 用户等级与积分系统
- 自定义头像上传
- 邮件密码重置

### 管理功能
- 用户状态管理（等待激活/活跃/禁止）
- 标签管理与访问权限控制
- 评论删除与管理
- 站内消息通知

### 其他特性
- 黑暗模式支持，可自动切换
- SEO优化与sitemap
- RSS订阅支持
- 图片代理优化

## 积分规则

### 获取积分
- 签到：1-10随机积分
- 被点赞：+1积分
- 被评论：+1积分
- 发布评论：+1积分
- 发布帖子：1-5随机积分

### 消耗积分
- 生成邀请码：-50积分
- 帖子被删除：-5积分
- 评论被删除：-3积分

## 数据库设计

| 表名 | 介绍 |
|------|------|
| tb_comment | 评论详情表 |
| tb_inspect_log | 审核日志表 |
| tb_invite_record | 邀请码表 |
| tb_message | 消息表 |
| tb_post | 文章表 |
| tb_post_tag | 文章标签关系表 |
| tb_settings | 系统设置表 |
| tb_statistics | 数据统计表 |
| tb_tag | 标签详情表 |
| tb_user | 用户表 |
| tb_vote | 投票表 |

## 快速开始

### 环境要求
- Go 1.16+
- PostgreSQL
- SMTP服务（可选，用于邮件功能）
- Google OAuth（可选，用于Google登录）

### 配置说明

在项目根目录创建 `.env` 文件，配置以下环境变量：

```dotenv
PORT=32919  # 服务端口
DB='host=127.0.0.1 user=postgres password=123456 dbname=hotnews port=5432 sslmode=disable TimeZone=Asia/Shanghai'  # 数据库连接
COOKIE_SECRET=test  # Cookie密钥
VERSION=0.0.1  # 版本号
SiteName=竹林  # 站点名称
SiteUrl=https://zhulink.vip  # 站点URL
EmailSender=发件人地址  # SMTP发件人
EmailSenderName=发件人姓名  # SMTP发件人名称
EmailPassword=邮箱密码  # SMTP密码
EmailSmtpHost=smtp服务器  # SMTP服务器
EmailSmtpPort=smtp端口  # SMTP端口
GIN_MODE=release  # 运行模式
ClientID=Google应用ID  # Google OAuth ID
ClientSecret=Google应用密钥  # Google OAuth Secret
```

### 启动说明

1. 本地开发启动：
```bash
go run main.go
```

2. Docker部署：
参考项目中的 `build.sh` 和 `docker-compose.yml` 文件

### 初始说明

- 首个注册用户自动成为管理员
- 管理员可以添加/管理父标签和子标签
- 支持用户列表管理和IP统计

## 开发计划

- [ ] 集市功能：支持用户发布虚拟商品，使用积分交易
- [ ] 帖子感谢功能：感谢消耗2积分，被感谢获得2积分
- [ ] 评论感谢功能：感谢消耗1积分，被感谢获得1积分

## 贡献

欢迎提交Issue和Pull Request。