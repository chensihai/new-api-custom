# New-API 部署指南（腾讯云轻量应用服务器）

> 适用环境：2核CPU / 4GB内存 / 5Mbps带宽 / Ubuntu 22.04 LTS

---

## 目录

1. [前置准备](#1-前置准备)
2. [安装 Docker](#2-安装-docker)
3. [系统参数调优](#3-系统参数调优)
4. [上传项目文件](#4-上传项目文件)
5. [配置环境变量](#5-配置环境变量)
6. [启动服务](#6-启动服务)
7. [验证部署](#7-验证部署)
8. [常用运维命令](#8-常用运维命令)
9. [故障排查](#9-故障排查)

---

## 1. 前置准备

### 1.1 购买服务器

在腾讯云控制台购买轻量应用服务器：
- **镜像**：Ubuntu 22.04 LTS 64位
- **规格**：2核CPU / 4GB内存 / 5Mbps带宽
- **地域**：选择离你用户最近的地区

### 1.2 连接服务器

**Windows 用户**：
1. 下载 [PuTTY](https://www.putty.org/) 或使用 Windows Terminal
2. 使用 SSH 连接：
   ```
   ssh root@你的服务器IP
   ```
3. 输入密码（在腾讯云控制台重置密码后获得）

**Mac/Linux 用户**：
```bash
ssh root@你的服务器IP
```

### 1.3 更新系统

```bash
apt update && apt upgrade -y
```

---

## 2. 安装 Docker

### 2.1 一键安装 Docker

```bash
curl -fsSL https://get.docker.com | sh
```

等待安装完成（约 1-2 分钟）。

### 2.2 配置 Docker 日志限制

防止日志文件占满磁盘：

```bash
mkdir -p /etc/docker

cat > /etc/docker/daemon.json << 'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2"
}
EOF
```

### 2.3 启动 Docker

```bash
systemctl restart docker
systemctl enable docker
```

### 2.4 验证安装

```bash
docker --version
```

看到类似 `Docker version 24.x.x` 表示安装成功。

---

## 3. 系统参数调优

针对 2核4G 小规格服务器进行优化，提升并发能力。

### 3.1 配置 TCP 连接参数

```bash
cat > /etc/sysctl.d/99-new-api-tuning.conf << 'EOF'
# TCP 连接优化
net.core.somaxconn = 1024
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15
net.ipv4.tcp_keepalive_time = 600
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.tcp_keepalive_probes = 5

# 文件描述符
fs.file-max = 65536

# 内存策略
vm.swappiness = 10
EOF
```

### 3.2 应用配置

```bash
sysctl --system
```

### 3.3 配置文件描述符限制

```bash
cat > /etc/security/limits.d/99-new-api.conf << 'EOF'
* soft nofile 65536
* hard nofile 65536
root soft nofile 65536
root hard nofile 65536
EOF
```

---

## 4. 上传项目文件

### 4.1 创建项目目录

```bash
mkdir -p /opt/new-api
cd /opt/new-api
```

### 4.2 克隆项目仓库

本项目使用自定义修改的代码，需要从自己的 GitHub 仓库克隆：

```bash
apt install git -y

# 克隆你的自定义仓库（替换为你的仓库地址）
git clone https://github.com/chensihai/new-api-custom.git .

# 或者使用 SSH 方式（推荐，更安全）
# git clone git@github.com:chensihai/new-api-custom.git .
```

> **说明**：克隆后会在 `/opt/new-api` 目录下得到完整源码，包括：
> - `Dockerfile` — 用于构建自定义镜像
> - `docker-compose-mysql.yml` — MySQL 部署配置
> - `web/` — 前端源码
> - Go 后端源码

---

## 5. 配置环境变量

### 5.1 创建 .env 文件

```bash
cd /opt/new-api

cat > .env << 'EOF'
# ========== 请修改以下密码 ==========
# MySQL root 密码（建议 16 位以上，包含字母数字符号）
MYSQL_ROOT_PASSWORD=YourStrongPassword123!@#

# Redis 密码
REDIS_PASSWORD=YourRedisPassword456!@

# Session 密钥（随机字符串，32位以上）
SESSION_SECRET=ChangeThisToRandomString32Chars

# ========== 以下通常不需要修改 ==========
# 时区
TZ=Asia/Shanghai

# 是否启用手机号认证
PHONE_AUTH_ENABLED=false
EOF
```

### 5.2 生成随机密码（可选）

如果不知道怎么设置密码，可以用这个命令生成：

```bash
# 生成 32 位随机密码
openssl rand -base64 32
```

把生成的结果替换到 `.env` 文件中。

---

## 6. 启动服务

### 6.1 选择部署模式

| 模式 | 配置文件 | 数据库 | 适用场景 |
|------|---------|--------|----------|
| MySQL 模式 | docker-compose-mysql.yml | MySQL 8.0 | **推荐**，生产环境 |
| PostgreSQL 模式 | docker-compose-prod.yml | PostgreSQL 15 | 需要高级特性 |

### 6.2 构建自定义镜像

由于使用自定义修改的代码，需要在服务器上构建 Docker 镜像：

```bash
cd /opt/new-api

# 构建镜像（约 3-5 分钟，首次会下载依赖）
docker build -t new-api:latest .
```

构建过程说明：
1. **第一阶段**：编译前端（使用 Bun）
2. **第二阶段**：编译后端（使用 Go）
3. **第三阶段**：打包最终镜像

> **国内网络加速**：如遇 Docker Hub 拉取超时，可配置镜像加速：
> ```bash
> # 编辑 Docker 配置（已有 daemon.json 则追加）
> cat >> /etc/docker/daemon.json << 'EOF'
> {
>   "registry-mirrors": [
>     "https://docker.1ms.run",
>     "https://docker.xuanyuan.me"
>   ]
> }
> EOF
> systemctl restart docker
> ```

### 6.3 启动服务（MySQL 模式）

```bash
cd /opt/new-api

docker compose -f docker-compose-mysql.yml up -d
```

看到类似输出表示启动成功：

```
[+] Running 4/4
 ✔ Network new-api-network  Created
 ✔ Container mysql          Started
 ✔ Container redis          Started
 ✔ Container new-api-mysql  Started
```

### 6.4 等待服务就绪

首次启动需要初始化数据库，大约等待 30-60 秒。

---

## 7. 验证部署

### 7.1 检查容器状态

```bash
docker compose -f docker-compose-mysql.yml ps
```

所有服务的 `STATUS` 应该是 `Up` 或 `Up (healthy)`。

### 7.2 检查服务健康

```bash
curl http://localhost:3000/api/status
```

返回 `{"success":true,...}` 表示服务正常。

### 7.3 访问 Web 界面

在浏览器中访问：

```
http://你的服务器公网IP:3000
```

**首次访问需要注册管理员账号**：
1. 点击"注册"
2. 输入用户名和密码
3. 注册成功后即可登录

### 7.4 配置防火墙（重要！）

在腾讯云控制台：
1. 进入服务器详情页
2. 点击"防火墙"标签
3. 添加规则：
   - 协议：TCP
   - 端口：3000
   - 策略：允许

---

## 8. 常用运维命令

### 8.1 查看日志

```bash
# 查看所有服务日志
docker compose -f docker-compose-mysql.yml logs

# 实时查看新 API 日志
docker compose -f docker-compose-mysql.yml logs -f new-api

# 查看最近 100 行
docker compose -f docker-compose-mysql.yml logs --tail 100 new-api
```

### 8.2 重启服务

```bash
# 重启所有服务
docker compose -f docker-compose-mysql.yml restart

# 只重启 new-api
docker compose -f docker-compose-mysql.yml restart new-api
```

### 8.3 停止服务

```bash
docker compose -f docker-compose-mysql.yml down
```

### 8.4 更新版本

当 GitHub 仓库有新代码更新时：

```bash
cd /opt/new-api

# 1. 拉取最新代码
git pull origin dev  # 或 main，根据你的分支

# 2. 重新构建镜像
docker build -t new-api:latest .

# 3. 重启服务使用新镜像
docker compose -f docker-compose-mysql.yml up -d
```

> **提示**：构建过程约 3-5 分钟，期间服务仍可正常运行，新镜像构建完成后会自动切换。

### 8.5 查看资源使用

```bash
docker stats
```

---

## 9. 故障排查

### 问题 1：容器启动后立即退出

**检查日志**：
```bash
docker compose -f docker-compose-mysql.yml logs new-api
```

**常见原因**：
- MySQL 还没就绪：等待 30 秒后重试
- 密码配置错误：检查 `.env` 文件

### 问题 2：访问 3000 端口无响应

**检查步骤**：

```bash
# 1. 检查容器是否运行
docker ps

# 2. 检查端口是否监听
netstat -tlnp | grep 3000

# 3. 检查防火墙
# 腾讯云控制台 → 防火墙 → 确认 3000 端口已放行
```

### 问题 3：内存不足 (OOM)

**症状**：容器被强制终止，日志有 `OOMKilled`

**解决方案**：

1. 查看内存使用：
```bash
docker stats --no-stream
free -h
```

2. 调整 MySQL 内存参数（已优化，如仍不够可继续调小）：
```bash
# 编辑 docker-compose-mysql.yml
# 将 innodb-buffer-pool-size=512M 改为 256M
```

### 问题 4：磁盘空间不足

**检查磁盘**：
```bash
df -h
```

**清理 Docker 缓存**：
```bash
# 清理未使用的镜像
docker image prune -a

# 清理未使用的卷
docker volume prune
```

### 问题 5：MySQL 连接失败

**检查 MySQL 状态**：
```bash
docker compose -f docker-compose-mysql.yml logs mysql
```

**进入 MySQL 容器检查**：
```bash
docker exec -it mysql mysql -uroot -p你的密码 -e "SHOW DATABASES;"
```

---

## 附录 A：配置 HTTPS（可选但推荐）

使用 Nginx + Let's Encrypt 配置 HTTPS。

### A.1 安装 Nginx

```bash
apt install nginx certbot python3-certbot-nginx -y
```

### A.2 配置 Nginx

```bash
cat > /etc/nginx/sites-available/new-api << 'EOF'
server {
    listen 80;
    server_name 你的域名.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 支持
        proxy_buffering off;
        proxy_cache off;
    }
}
EOF

ln -s /etc/nginx/sites-available/new-api /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

### A.3 申请 SSL 证书

```bash
certbot --nginx -d 你的域名.com
```

按提示输入邮箱，同意条款即可。

### A.4 自动续期

```bash
certbot renew --dry-run
```

Certbot 会自动添加续期定时任务。

---

## 附录 B：资源限制说明

Docker Compose 已配置资源限制，防止单个服务吃满内存：

| 服务 | CPU 限制 | 内存限制 | 说明 |
|------|----------|----------|------|
| new-api | 1 核 | 512 MB | Go 应用，实际占用约 100-200 MB |
| MySQL | 0.8 核 | 1536 MB | 数据库，实际占用约 600-800 MB |
| Redis | 0.3 核 | 256 MB | 缓存，实际占用约 100-200 MB |

**总计**：约 1.5-2 GB，为系统预留 2 GB。

---

## 附录 C：安全建议

1. **修改默认密码**：`.env` 文件中的密码必须修改，登录后立即修改 Web 管理员密码
2. **限制端口暴露**：仅开放 3000 端口，数据库端口（3306、5432、6379）不对外开放
3. **Nginx 反代安全**：配置 Nginx 后，关闭公网 3000 端口，仅开放 80/443
4. **定期备份**：使用 `docker exec mysql mysqldump` 备份数据库
5. **定期更新**：定期 `git pull` 拉取代码并重新构建镜像
6. **配置 HTTPS**：生产环境强烈建议使用 HTTPS，防止密码被窃听

---

## 联系支持

如有问题，请在 GitHub 提交 Issue：
https://github.com/chensihai/new-api-custom/issues
