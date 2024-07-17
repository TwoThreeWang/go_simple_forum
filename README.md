### [Go开发的极简论坛](https://one.wangtwothree.com)

基于 [https://github.com/kingwrcy/hotnews](https://github.com/kingwrcy/hotnews) 修改而来

在源代码的基础上增改的功能：

- [x] 发布及评论的编辑器改为 md 编辑器，文章内容保存为 md 格式
- [x] 优化文章内容行间距
- [x] 评论及文章内容链接改为新标签页打开
- [x] 头像可以设置自定义头像
- [x] 用户信息修改功能
- [x] 邮件重置密码功能
- [x] 增加404页面
- [ ] 增加积分功能
- [ ] 增加签到功能
- [x] 标签增加对游客隐藏选项
- [x] 新注册用户邮箱激活功能
- [x] 管理员管理用户状态（Wait：等待激活；Active：活跃用户；Banned：禁止用户）
- [x] 管理员管理文章状态（Wait：等待审核；Active：正常；Rejected：删除）

[Docker镜像](https://hub.docker.com/repository/docker/kingwrcy/hotnews)

| 环境变量              | 解释                  | 示例                                                                                                             |
|-------------------|---------------------|----------------------------------------------------------------------------------------------------------------|
| PORT              | 监听端口                | 选填,默认32919                                                                                                     |
| COOKIE_SECRET     | cookie密钥            | 必填,如:UbnpjqcvDJ8mDCB                                                                                           |
| STATIC_CDN_PREFIX | 静态资源CDN前缀           | 选填,默认取使用本地静态文件                                                                                                 |
| DB                | 数据库链接,目前只支持Postgres | 必填,'host=localhost user=username password=password dbname=hn port=5432 sslmode=disable TimeZone=Asia/Shanghai' |

默认第一个注册的用户是管理员,自行注册即可.

目前可管理的功能很少,唯一能做的就是添加父标签/子标签,设置标签颜色等.

后台带了个用户列表和ip统计等.

需要的朋友自行部署吧.

得意于强大的go的内嵌静态资源的功能,镜像包只有**6.29mb**,启动之后占用内存只有**28mb**.

极度适合小内存的机器.当然数据库另说.

![alt](https://openai-75050.gzc.vod.tencent-cloud.com/openaiassets_5ba4ebcbd2030fee5ac43c38e41a0f41_2579861720144999302.png 'title')

数据库表介绍

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

### docker启动

1. 随便着个目录,在这个目录底下新建`.env`文件,内容如下,每个字段含义上面有写
```dotenv
PORT=32919
DB='host=localhost user=postgres password=a123456 dbname=hn port=5432 sslmode=disable TimeZone=Asia/Shanghai'
COOKIE_SECRET=UbnpjqcvDJ8mDCB
```

2. 使用如下命令启动
```shell
docker run --name hotnews -d --env-file .env -p 32912:32912 kingwrcy/hotnews:latest
```

3. 打开浏览器访问`本地ip:32912`即可.