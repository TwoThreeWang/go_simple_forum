### [Go开发的极简论坛](https://github.com/TwoThreeWang/go_simple_forum)

基于 [https://github.com/kingwrcy/hotnews](https://github.com/kingwrcy/hotnews) 修改而来，感谢原作者的辛苦开发。

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
- [ ] 增加签到功能
- [ ] 积分兑换邀请码

#### 关于积分增减规则

- 签到可以获取积分，普通签到获取1积分，试试手气获取1-10随机积分
- 被点赞+1积分
- 被评论+1积分
- 生成邀请码-50积分
- 帖子被删除-5积分
- 评论被删除-3积分

### Demo

竹林：[https://zhulink.wangtwothree.com](https://zhulink.wangtwothree.com)

### 说明

默认第一个注册的用户是管理员，启动后自行注册即可.

目前可管理的功能很少,唯一能做的就是添加父标签/子标签,设置标签颜色等.

后台带了个用户列表和ip统计等.

得意于强大的 go 的内嵌静态资源的功能,镜像包只有**6.29mb**,启动之后占用内存只有**28mb**.

极度适合小内存的机器.当然数据库另说.

![alt](https://openai-75050.gzc.vod.tencent-cloud.com/openaiassets_5ba4ebcbd2030fee5ac43c38e41a0f41_2579861720144999302.png 'title')

### 数据库表介绍

| 表名                |  介绍     |
|-------------------|---------|
| tb_comment        | 评论详情表   |
| tb_inspect_log    | 审核日志表   |
| tb_invite_record  | 邀请码表    |
| tb_message        | 消息表     |
| tb_post           | 文章表     |
| tb_post_tag       | 文章标签关系表 |
| tb_settings       | 系统设置表   |
| tb_statistics     | 数据统计表   |
| tb_tag            | 标签详情表   |
| tb_user           | 用户表     |
| tb_vote           | 投票表     |

### 环境变量

| 环境变量              | 解释                                                                               | 示例                                                                                                             |
|-------------------|----------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------|
| PORT              | 监听端口                                                                             | 选填,默认32919                                                                                                     |
| COOKIE_SECRET     | cookie密钥                                                                         | 必填,如:UbnpjqcvDJ8mDCB                                                                                           |
| STATIC_CDN_PREFIX | 静态资源CDN前缀                                                                        | 选填,默认取使用本地静态文件                                                                                                 |
| DB                | 数据库链接,目前只支持Postgres                                                              | 必填,'host=localhost user=username password=password dbname=hn port=5432 sslmode=disable TimeZone=Asia/Shanghai' |
| VERSION           | 程序版本号                                                                            | 必填, 0.0.1                                                                                                      |
| SiteName          | 网站名称                                                                             | 必填, 竹林                                                                                                         |
| SiteUrl           | 网站链接                                                                             | 必填, https://zhulink.club                                                                                       |
| EmailApiUrl       | 邮件发送链接，邮件发送使用的是cloudflare，相关教程 https://wangtwothree.com/sao/cloudflare-mail.html | 必填（不填影响邮件发送功能，其他功能正常）, https://<sender-name>.<your-name>.workers.dev                                           |
| EmailSender       | 发件人邮箱                                                                            | 必填（不填影响邮件发送功能，其他功能正常）, 竹林                                                                                      |
| EmailSenderName   | 发件人名称                                                                            | 必填（不填影响邮件发送功能，其他功能正常）, 竹林                                                                                      |
| GIN_MODE          | GIN程序运行模式，debug调试模式、release生产模式、test测试模式                                         | 选填, release                                                                                                    |


### 启动

1. 程序目录下新建`.env`文件,内容如下,每个字段含义上面有写
```dotenv
PORT=32919
DB='host=127.0.0.1 user=postgres password=123456 dbname=hotnews port=5432 sslmode=disable TimeZone=Asia/Shanghai'
COOKIE_SECRET=test
VERSION=0.0.1
SiteName=竹林
SiteUrl=https://zhulink.club
EmailApiUrl=https://<sender-name>.<your-name>.workers.dev
EmailSender=发件人地址
EmailSenderName=发件人姓名
GIN_MODE=release
```

2. 启动

本地启动直接运行 main.go 即可

build.sh 文件中有镜像打包及容器启动命令

3. 打开浏览器访问`本地ip:32912`即可.