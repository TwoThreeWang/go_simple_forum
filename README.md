# Go Simple Forum

本项目不再更新，转移到：[https://github.com/TwoThreeWang/zhulink](https://github.com/TwoThreeWang/zhulink)

一个简洁轻量的 Go 语言开发的论坛系统，采用前后端不分离架构，提供完整的社区讨论功能。

本项目基于 [https://github.com/kingwrcy/hotnews](https://github.com/kingwrcy/hotnews) 二开，感谢原作者的贡献。

## Demo

竹林：[https://zhulink.vip](https://zhulink.vip)

![论坛截图](https://openai-75050.gzc.vod.tencent-cloud.com/openaiassets_5ba4ebcbd2030fee5ac43c38e41a0f41_2579861720144999302.png)

## 🌟 功能特性

### 核心功能
- **用户系统**：注册、登录、个人资料管理
- **帖子管理**：发布、编辑、删除、搜索
- **评论系统**：支持多级评论回复
- **标签系统**：灵活的帖子分类和标签管理
- **投票系统**：点赞、踩功能
- **消息通知**：实时消息提醒
- **内容审核**：举报和管理员审核机制

### 管理功能
- **用户管理**：用户列表、权限管理
- **内容审核**：帖子/评论审核
- **统计分析**：用户活跃度、帖子统计
- **系统设置**：网站配置管理

### 技术特色
- **极简设计**：得益于Go语言强大的内嵌静态资源功能，镜像包仅6.29MB，运行时内存占用仅28MB
- **高性能**：基于 Go 的高并发处理
- **缓存优化**：使用内存缓存提升访问速度
- **SEO友好**：支持搜索引擎优化
- **响应式设计**：适配移动端和桌面端
- **Markdown支持**：支持完整的 Markdown 格式

## 🛠️ 技术栈

### 后端
- **语言**：Go 1.19+
- **框架**：Gin Web Framework
- **数据库**：PostgreSQL
- **ORM**：GORM
- **验证**：Validator
- **日志**：Zap

### 前端
- **模板引擎**：Go HTML Template
- **样式**：UnoCSS + 自定义CSS
- **脚本**：jQuery + 原生JavaScript
- **编辑器**：Markdown编辑器（支持ScEditor）
- **图标**：自定义SVG图标

### 部署
- **容器化**：Docker + Docker Compose
- **反向代理**：Nginx
- **进程管理**：Supervisor

## 🚀 快速开始

### 环境要求
- Go 1.19 或更高版本
- PostgreSQL 13 或更高版本
- SMTP服务（可选，用于邮件功能）
- Google OAuth（可选，用于Google登录）

### 安装步骤

1. **克隆项目**
```bash
git clone https://github.com/TwoThreeWang/go_simple_forum
cd go_simple_forum
```

2. **安装依赖**
```bash
go mod download
```

3. **配置环境**
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置文件
vim .env
```

4. **启动应用**
```bash
./bulid.sh
```

6. **访问应用**
打开浏览器访问 `http://localhost:32919`

### 初始说明

- 首个注册用户自动成为管理员
- 管理员可以添加/管理父标签和子标签
- 支持用户列表管理和IP统计

## 📁 项目结构

```
go_simple_forum/
├── handler/          # 路由处理器
├── model/           # 数据模型
├── middleware/      # 中间件
├── templates/       # HTML模板
├── static/          # 静态资源
├── utils/           # 工具函数
├── provider/        # 服务提供者
├── task/            # 定时任务
├── sql/             # 数据库脚本
├── vo/              # 视图模型
├── Dockerfile       # Docker配置
├── docker-compose.yml
├── go.mod          # Go模块配置
└── main.go         # 程序入口
```

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

## 🔧 开发指南

### 开发命令
```bash
# 运行应用
go run main.go

# 运行测试
go test ./...

# 代码格式化
go fmt ./...

# 依赖整理
go mod tidy
```

### 添加新功能
1. 在 `model/` 目录定义数据模型
2. 在 `handler/` 目录添加处理器
3. 在 `templates/` 目录创建对应模板
4. 更新路由配置

## 📊 性能优化

### 缓存策略
- **页面缓存**：热门页面缓存5分钟
- **用户缓存**：用户信息缓存30分钟
- **统计缓存**：统计数据缓存1小时
- **搜索结果**：搜索结果缓存15分钟

### 数据库优化
- 关键字段索引优化
- 查询语句优化
- 分页查询使用游标

### 前端优化
- 静态资源压缩
- CDN加速
- 懒加载图片
- 浏览器缓存

## 🔐 安全特性

- **密码安全**：bcrypt加密存储
- **CSRF保护**：所有表单包含CSRF令牌
- **XSS防护**：输入内容严格过滤
- **SQL注入防护**：使用预编译语句
- **验证码**：集成Cloudflare Turnstile
- **频率限制**：API调用频率限制

## 📄 许可证

本项目采用 MIT 许可证。

### 更新日志
查看 [CHANGELOG.md](CHANGELOG.md) 了解版本更新历史。

## 🙏 致谢

- 感谢所有贡献者的努力
- 感谢开源社区的支持
- 特别鸣谢使用到的开源项目

---

**⭐ 如果这个项目对你有帮助，请给个星标支持一下！**
