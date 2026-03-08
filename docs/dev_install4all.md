# HL6 开发环境搭建指南

> 本文面向所有技术水平的开发者，帮助你从零搭建 HL6 的本地开发环境并成功运行 `make dev`。
>
> 源码仓库：`https://git.houlang.cloud/houlangcloud/hl6`

---

## 导航：找到适合你的起点

| 你的情况 | 建议起点 |
|---------|---------|
| 第一次接触编程/命令行 | 从 [第一章：前置概念](#1-前置概念理解) 开始 |
| 会用命令行，但没搞过 Go/Node.js | 跳到 [第二章：安装依赖](#2-安装开发依赖) |
| 熟悉全栈开发，只需快速上手 | 跳到 [速查清单](#速查清单老手专用) |
| 在服务器/无桌面环境部署 | 跳到 [第五章：服务器环境](#5-服务器无桌面环境) |
| 遇到问题了 | 跳到 [第六章：常见错误排查](#6-常见错误排查) |

---

## 速查清单（老手专用）

已经熟悉 Go、Node.js、Docker？直接照着做：

```bash
# 1. 克隆
git clone https://git.houlang.cloud/houlangcloud/hl6.git && cd hl6

# 2. 环境变量
cp .env.example .env
# 编辑 .env，填写 OIDC 等必要配置（SESSION_SECRET 可留空自动生成）
# 可选：若要加密 Cloudflare Token，再填写 ENCRYPTION_KEY

# 3. 安装前端依赖
cd web && npm install && cd ..

# 4. 启动（PostgreSQL + Go 后端 + Vite 前端）
make dev
```

**最低版本要求：**
- Go ≥ 1.25
- Node.js ≥ 22（推荐 LTS）
- Docker & Docker Compose（用于 PostgreSQL 16）
- Git
- Make

---

## 1. 前置概念理解

> 如果你已经知道"终端"、"环境变量"、"端口"这些概念是什么，可以直接跳到[第二章](#2-安装开发依赖)。

### 1.1 HL6 是什么

HL6 是一个域名/子域名管理平台。它由两部分组成：

- **前端**（`web/` 目录）：用户看到的网页界面，用 React + TypeScript 写成
- **后端**（`server/` 目录）：处理业务逻辑的服务器程序，用 Go 语言写成

还需要一个**数据库**（PostgreSQL）来存储数据。本项目使用 Docker 容器来运行数据库，这样你不需要手动安装 PostgreSQL。

### 1.2 关键概念速览

| 概念 | 一句话解释 |
|------|-----------|
| **终端 / Terminal** | 你输入命令让电脑执行操作的文本界面。Windows 上叫 PowerShell 或 CMD，macOS/Linux 上叫 Terminal |
| **Git** | 代码版本管理工具，用来下载（"克隆"）和管理源代码 |
| **Docker** | 一种轻量级虚拟化技术，让你不用手动安装数据库等软件，而是运行预配置好的"容器" |
| **环境变量** | 告诉程序运行时配置信息的键值对，本项目统一写在 `.env` 文件中 |
| **端口** | 程序监听网络请求的"门牌号"。前端用 5173，后端用 8080，数据库用 5432 |
| **Make** | 任务自动化工具，`make dev` 就是一条命令同时启动数据库、后端和前端 |
| **OIDC** | 开放身份认证协议，HL6 通过它实现用户登录（需要一个 OIDC 提供商，如 Logto、Keycloak 等） |
| **npm** | Node.js 的包管理器，用于安装前端依赖库 |

### 1.3 整体启动流程

```
make dev 实际上做了三件事（并行）：

1. docker compose up -d     → 启动 PostgreSQL 数据库容器
2. go run ./cmd/server      → 编译并启动 Go 后端 (端口 8080)
3. npm run dev (vite)       → 启动前端开发服务器 (端口 5173)

打开浏览器访问 http://localhost:5173 即可看到页面。
前端会自动将 /api 请求代理到后端 8080 端口。
```

---

## 2. 安装开发依赖

### 2.1 操作系统对照表

| 依赖 | macOS | Windows | Ubuntu/Debian | Fedora/RHEL | Arch Linux |
|------|-------|---------|---------------|-------------|------------|
| Git | Xcode CLT 自带 | [git-scm.com](https://git-scm.com) | `apt install git` | `dnf install git` | `pacman -S git` |
| Make | Xcode CLT 自带 | 见 [2.7 节](#27-windows-专项) | `apt install make` | `dnf install make` | `pacman -S make` |
| Docker | Docker Desktop | Docker Desktop | 见下文 | 见下文 | `pacman -S docker docker-compose` |
| Go | brew / 官网 | 官网安装包 | 见下文 | 见下文 | `pacman -S go` |
| Node.js | fnm / nvm | fnm / nvm-windows | fnm / nvm | fnm / nvm | `pacman -S nodejs npm` |

### 2.2 安装 Git

**macOS：**
```bash
# 输入任意 git 命令即可触发 Xcode Command Line Tools 安装提示
git --version
# 如果弹出安装对话框，点击"安装"即可
```

**Windows：**
1. 访问 https://git-scm.com/download/win 下载安装包
2. 安装时一路默认即可，建议勾选 "Git Bash Here"
3. 安装后打开 "Git Bash" 验证：`git --version`

**Linux（Debian/Ubuntu）：**
```bash
sudo apt update && sudo apt install -y git
```

**Linux（Fedora/RHEL）：**
```bash
sudo dnf install -y git
```

### 2.3 安装 Docker

Docker 用于运行 PostgreSQL 16 数据库，不需要手动安装 PostgreSQL。

**macOS：**
```bash
# 推荐使用 Homebrew
brew install --cask docker
# 安装后启动 Docker Desktop 应用
```

或从 https://www.docker.com/products/docker-desktop/ 下载安装。

**Windows：**
1. 下载 Docker Desktop：https://www.docker.com/products/docker-desktop/
2. 安装并重启电脑
3. 启动 Docker Desktop，等待 Docker Engine 就绪
4. 在终端中验证：`docker --version`

> **Windows 注意事项**：Docker Desktop 需要 WSL 2 或 Hyper-V。安装过程中会提示你启用 WSL 2，按提示操作即可。如果此前未安装 WSL，可能需要重启。

**Linux（Ubuntu/Debian）：**
```bash
# 安装 Docker Engine（官方推荐方式）
sudo apt update
sudo apt install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 允许当前用户免 sudo 使用 docker
sudo usermod -aG docker $USER
newgrp docker
```

> **中国大陆用户**：如果 `download.docker.com` 访问缓慢，可使用镜像源：
> ```bash
> # 替换上面 curl 和 echo 命令中的 download.docker.com 为以下任一镜像：
> # - mirrors.aliyun.com/docker-ce
> # - mirrors.tuna.tsinghua.edu.cn/docker-ce
> ```
>
> Docker Hub 镜像加速器配置（`/etc/docker/daemon.json`）：
> ```json
> {
>   "registry-mirrors": [
>     "https://docker.1ms.run",
>     "https://docker.xuanyuan.me"
>   ]
> }
> ```
> 配置后执行 `sudo systemctl restart docker`。
>
> 镜像加速器地址经常变动，如果以上不可用，请搜索"Docker 镜像加速器 2026"获取最新地址。

**Linux（Fedora/RHEL）：**
```bash
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
newgrp docker
```

### 2.4 安装 Go

本项目需要 **Go ≥ 1.25**。

**推荐方式 — 官网下载（全平台）：**

访问 https://go.dev/dl/ ，下载对应操作系统的安装包并安装。

**macOS（Homebrew）：**
```bash
brew install go
```

**Linux（手动安装，适用于所有发行版）：**
```bash
# 下载（替换版本号为最新版）
wget https://go.dev/dl/go1.25.5.linux-amd64.tar.gz

# 安装
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.5.linux-amd64.tar.gz

# 添加到 PATH（写入 shell 配置文件）
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

> **ARM 架构**（如 Apple Silicon Mac 原生 Linux、树莓派）：下载文件名中 `amd64` 替换为 `arm64`。

**中国大陆用户**——设置 Go 模块代理（**重要**，否则 `go run` 下载依赖会非常慢或超时）：
```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

验证安装：
```bash
go version
# 应输出 go1.25.x 或更高
```

### 2.5 安装 Node.js

本项目需要 **Node.js ≥ 22**。

**推荐方式 — 使用 fnm（全平台版本管理器）：**

fnm（Fast Node Manager）是一个跨平台的 Node.js 版本管理器，比 nvm 更快。

```bash
# macOS / Linux
curl -fsSL https://fnm.vercel.app/install | bash
# 重新打开终端或 source 你的 shell 配置文件

# Windows（PowerShell）
winget install Schniz.fnm

# 安装 Node.js
fnm install 22
fnm use 22
fnm default 22
```

> **中国大陆用户**：fnm 安装脚本如果访问慢，可改用 Homebrew（`brew install fnm`）或从 [fnm GitHub Releases](https://github.com/Schniz/fnm/releases) 手动下载二进制文件。

**备选方式 — nvm（macOS / Linux）：**
```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash
# 重新打开终端
nvm install 22
nvm use 22
nvm alias default 22
```

**备选方式 — 官网直装（不推荐长期使用，适合快速体验）：**

访问 https://nodejs.org/ 下载 LTS 版本安装。

> **中国大陆用户**——设置 npm 镜像（**重要**，否则 `npm install` 会非常慢）：
> ```bash
> npm config set registry https://registry.npmmirror.com
> ```

验证安装：
```bash
node --version   # 应输出 v22.x 或更高
npm --version    # 应输出 10.x 或更高
```

### 2.6 安装 Make

Make 通常在 macOS 和 Linux 上已预装。

**macOS：**
```bash
# 通常随 Xcode Command Line Tools 安装
# 如果没有：
xcode-select --install
```

**Linux：** 大多数发行版自带。如果没有：
```bash
# Debian/Ubuntu
sudo apt install -y make

# Fedora/RHEL
sudo dnf install -y make
```

**Windows：** 见下一节。

### 2.7 Windows 专项

Windows 上的开发体验与 macOS/Linux 有差异，以下是几个关键点：

#### 推荐使用 WSL 2（Windows Subsystem for Linux）

WSL 2 让你在 Windows 上运行完整的 Linux 环境，**强烈推荐**用于本项目的开发。在 WSL 内，所有 Linux 的安装步骤都直接适用。

```powershell
# 以管理员身份打开 PowerShell
wsl --install
# 默认安装 Ubuntu，安装后重启电脑
# 重启后打开 "Ubuntu" 应用，设置用户名和密码
```

在 WSL 内，按照上面 Linux（Ubuntu/Debian）的步骤安装所有依赖即可。Docker Desktop 会自动与 WSL 集成。

#### 不使用 WSL 的情况

如果坚持在 Windows 原生环境开发：

1. **Make**：安装 [Chocolatey](https://chocolatey.org/install) 后运行 `choco install make`，或使用 [GnuWin32](https://gnuwin32.sourceforge.net/packages/make.htm)。也可以不用 `make dev`，而是手动分三个终端窗口分别运行：
   ```powershell
   # 终端 1
   docker compose up -d
   
   # 终端 2
   cd server
   go run ./cmd/server
   
   # 终端 3
   cd web
   npm run dev
   ```

2. **行尾符号**：Git 克隆时确保配置正确：
   ```bash
   git config --global core.autocrlf input
   ```

3. **路径长度限制**：Node.js 的 `node_modules` 路径可能很深，建议开启长路径支持：
   ```powershell
   # 以管理员身份运行 PowerShell
   New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" `
     -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force
   ```

---

## 3. 克隆代码与启动

### 3.1 克隆仓库

```bash
git clone https://git.houlang.cloud/houlangcloud/hl6.git
cd hl6
```

> **中国大陆用户**：本仓库托管在自部署 GitLab（`git.houlang.cloud`），不需要代理即可访问。如果遇到网络问题，请检查 DNS 设置或尝试使用其他 DNS（如 `223.5.5.5`）。

### 3.2 配置环境变量

```bash
cp .env.example .env
```

用文本编辑器打开 `.env`，必须填写以下字段：

```env
# 数据库连接（使用 Docker 默认配置时无需修改）
DATABASE_URL=postgres://hl6:hl6dev@localhost:5432/hl6?sslmode=disable

# 服务器端口
SERVER_PORT=8080

# 管理员规则：首个注册用户会自动成为管理员

# OIDC 认证配置（必填）
OIDC_ISSUER=https://your-oidc-provider.example.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret

> **如何填写这三项？** 请参阅 [OIDC 提供商配置指南](./oidc.md)，其中包含 Logto、Casdoor、Keycloak、Authentik、Google、Microsoft Entra ID 等 9 种提供商的详细配置步骤和 Issuer 格式说明。

# Session 密钥（可留空；首次启动会自动生成并写入数据库）
SESSION_SECRET=

# 前端地址
FRONTEND_URL=http://localhost:5173
ALLOWED_ORIGINS=http://localhost:5173

# 加密密钥（可选：AES-256-GCM，32 字节十六进制；留空则 Cloudflare Token 明文存储）
ENCRYPTION_KEY=
```

如需启用 Cloudflare Token 加密，生成 `ENCRYPTION_KEY`：
```bash
openssl rand -hex 32
# 将输出复制粘贴到 .env 的 ENCRYPTION_KEY= 后面
```

> **没有 openssl？** Windows 上可以在 Git Bash 里运行。或者用 Python：
> ```bash
> python3 -c "import secrets; print(secrets.token_hex(32))"
> ```

> **没有 OIDC 提供商？** 你需要一个 OIDC 兼容的身份认证服务才能完成登录流程。推荐自部署 [Logto](https://logto.io/)（开源免费）或 [Casdoor](https://casdoor.org/)。完整的提供商配置教程见 **[OIDC 提供商配置指南](./oidc.md)**。
> **会话密钥说明**：服务会把内部会话密钥持久化在数据库 `system_configs._internal_session_secret`。数据库重置后会重新生成，会导致所有用户需要重新登录。

### 3.3 安装前端依赖

```bash
cd web
npm install
cd ..
```

> 首次 `npm install` 会下载约 200MB 的依赖包。中国大陆用户如果之前没有配置 npm 镜像，此步骤会很慢，请参考 [2.5 节](#25-安装-nodejs) 设置镜像。

### 3.4 启动

```bash
make dev
```

这条命令会：
1. 启动 PostgreSQL 容器（`docker compose up -d`）
2. 编译并启动 Go 后端（首次运行会自动下载 Go 依赖，建议提前配置 GOPROXY）
3. 启动 Vite 前端开发服务器

看到类似以下输出说明启动成功：
```
[+] Running 1/1
 ✔ Container hl6-postgres  Started
Database migrated successfully
OIDC provider discovered: issuer=...
Server starting on :8080
  VITE v7.x.x  ready in xxx ms
  ➜  Local:   http://localhost:5173/
```

打开浏览器访问 **http://localhost:5173** 即可。

### 3.5 停止服务

在运行 `make dev` 的终端按 `Ctrl+C` 即可停止前端和后端。

停止数据库容器：
```bash
make db-down
```

> 数据库数据存储在 Docker 卷 `pgdata` 中，`make db-down` 不会删除数据。下次 `make dev` 时数据仍在。如果要彻底清除数据：`docker compose down -v`。

---

## 4. 项目结构概览

```
hl6/
├── .env.example          # 环境变量模板
├── .env                  # 你的本地配置（不提交到 Git）
├── Makefile              # make dev 等快捷命令
├── docker-compose.yml    # PostgreSQL 容器定义
├── LICENSE               # AGPL-3.0 许可证
├── server/               # Go 后端
│   ├── cmd/server/       # 入口点 main.go
│   ├── internal/         # 内部包
│   │   ├── config/       # 环境变量加载
│   │   ├── handler/      # HTTP 处理器
│   │   ├── middleware/    # 认证、CORS、限流等中间件
│   │   ├── model/        # 数据库模型（GORM）
│   │   ├── repository/   # 数据访问层
│   │   ├── router/       # 路由配置
│   │   ├── service/      # 业务逻辑（Cloudflare DNS 等）
│   │   └── oidc/         # OIDC Discovery
│   └── go.mod            # Go 模块依赖
└── web/                  # React 前端
    ├── src/
    │   ├── pages/        # 路由页面
    │   ├── components/   # UI 和业务组件
    │   ├── hooks/        # 自定义 hooks（数据获取）
    │   ├── lib/          # API 客户端、工具函数
    │   ├── i18n/         # 国际化（6 种语言）
    │   └── types/        # TypeScript 类型定义
    ├── package.json      # 前端依赖
    └── vite.config.ts    # Vite 配置（代理 /api → :8080）
```

---

## 5. 服务器（无桌面）环境

在无桌面的 Linux 服务器上开发时，Docker Desktop 不可用，但 Docker Engine 完全可以在命令行下使用。

### 5.1 安装差异

- **Docker**：直接安装 Docker Engine（参考 [2.3 节](#23-安装-docker) Linux 部分），无需 Docker Desktop
- **编辑器**：使用 vim/nano 编辑 `.env`，或通过 VS Code Remote SSH 远程开发
- **端口访问**：
  - 如果从本地浏览器访问服务器上的 HL6，需要确保防火墙放行 5173 端口
  - 或使用 SSH 端口转发：
    ```bash
    ssh -L 5173:localhost:5173 -L 8080:localhost:8080 user@your-server
    ```
    然后在本地浏览器访问 `http://localhost:5173`

### 5.2 后台运行

开发环境不建议后台运行（需要看日志）。如果需要，可使用 `tmux` 或 `screen`：

```bash
# tmux 示例
tmux new -s hl6
make dev
# 按 Ctrl+B 然后按 D 脱离 session
# 重新连接：tmux attach -t hl6
```

### 5.3 低内存服务器

如果服务器内存 < 2GB：
- PostgreSQL 容器约占用 50-100MB
- Go 编译约需 500MB+
- Vite 开发服务器约需 200-300MB

可以分开启动来降低峰值内存占用：
```bash
make db-up                    # 先启动数据库
make dev-server &             # 后台启动后端
# 等待后端启动完成后
make dev-web                  # 再启动前端
```

---

## 6. 常见错误排查

### Docker 相关

**`docker: Cannot connect to the Docker daemon`**
- Docker 服务未启动。macOS/Windows：启动 Docker Desktop 应用。Linux：`sudo systemctl start docker`
- Linux 上如果不想加 `sudo`：`sudo usermod -aG docker $USER` 然后重新登录

**`port 5432 already in use`**
- 本机已有 PostgreSQL 在运行。停止它或修改 `docker-compose.yml` 端口映射：
  ```yaml
  ports:
    - "5433:5432"  # 改为 5433
  ```
  同时更新 `.env` 中 `DATABASE_URL` 的端口号

**`image not found: postgres:16-alpine`**（中国大陆常见）
- Docker Hub 拉取失败。配置 Docker 镜像加速器（见 [2.3 节](#23-安装-docker)）
- 或手动拉取：`docker pull mirrors.example.com/library/postgres:16-alpine`

### Go 后端相关

**`failed to connect database`**
- 数据库未启动。先运行 `make db-up`，等待几秒后再启动后端
- 检查 `.env` 中 `DATABASE_URL` 是否正确

**`OIDC discovery failed`**
- `.env` 中 `OIDC_ISSUER` 未配置或不可访问
- 确认 OIDC 提供商正在运行且网络可达
- 如果只是想看看前端界面，暂时无法绕过这一步——HL6 依赖 OIDC 完成认证

**`go: module ... downloading` 卡住或超时**
- 中国大陆用户未设置 Go 代理。运行：
  ```bash
  go env -w GOPROXY=https://goproxy.cn,direct
  ```
  然后重试

**`go: go.mod requires go >= 1.25`**
- Go 版本过低。运行 `go version` 检查，需要 ≥ 1.25。按 [2.4 节](#24-安装-go) 更新

### 前端相关

**`npm install` 报错或卡住**
- 检查 Node.js 版本：`node --version`，需要 ≥ 22
- 中国大陆用户设置镜像：`npm config set registry https://registry.npmmirror.com`
- 删除缓存重试：`rm -rf web/node_modules web/package-lock.json && cd web && npm install`

**`VITE vX.X.X ready` 但浏览器打开白屏**
- 打开浏览器开发者工具（F12）查看 Console 中的错误
- 常见原因：后端未启动导致 API 请求失败

**`port 5173 already in use`**
- 另一个 Vite 实例在运行。关掉或换端口：
  ```bash
  cd web && npx vite --port 5174
  ```

### Make 相关

**`make: command not found`**（Windows）
- Windows 原生不带 Make。使用 WSL 或手动分开运行命令（见 [2.7 节](#27-windows-专项)）

### 通用网络问题（中国大陆）

| 操作 | 加速方案 |
|------|---------|
| `git clone` | 本仓库在自部署 GitLab，通常无需代理 |
| `go mod download` | `go env -w GOPROXY=https://goproxy.cn,direct` |
| `npm install` | `npm config set registry https://registry.npmmirror.com` |
| `docker pull` | 配置 Docker 镜像加速器（见 [2.3 节](#23-安装-docker)） |
| 下载 Go / Node.js 安装包 | 使用 npmmirror 提供的镜像：`https://registry.npmmirror.com/binary.html` |

---

## 7. 善用 AI 工具

开发过程中遇到的大多数问题（环境配置、报错排查、代码理解），AI 工具都能提供快速帮助。

### 7.1 推荐工具

| 工具 | 适合场景 |
|------|---------|
| [Claude Code](https://claude.ai/code)（CLI） | 在终端中直接问问题、让 AI 阅读代码库并修改代码 |
| [Cursor](https://cursor.com) / [Windsurf](https://windsurf.com) | 带 AI 辅助的代码编辑器，适合日常开发 |
| [Claude](https://claude.ai) / [ChatGPT](https://chat.openai.com) | 通用对话，适合概念解释和问题排查 |

### 7.2 高效提问技巧

当你遇到报错时，向 AI 提问应该包含：

1. **完整的错误信息**（复制粘贴，不要截图）
2. **你在做什么操作**（哪条命令）
3. **你的操作系统和相关工具版本**

示例：

> 我在 Ubuntu 24.04 上运行 `make dev`，Go 后端启动时报错：
> ```
> failed to connect database: dial tcp 127.0.0.1:5432: connect: connection refused
> ```
> Docker 容器状态是 `Up 3 seconds (health: starting)`。

这样的提问 AI 能直接给出有效回答（数据库还没初始化完，等 health check 通过即可）。

### 7.3 让 AI 帮你理解代码

如果你刚接触本项目，可以直接问 AI：

- "解释一下 `server/cmd/server/main.go` 的启动流程"
- "前端 `hooks/use-auth.ts` 的认证逻辑是怎样的？"
- "帮我看看这个 API 请求失败的原因"（附上浏览器 Network 面板截图或请求详情）

使用 Claude Code 时，它可以直接读取项目文件，不需要你手动复制代码。

### 7.4 用 AI 编程助手帮你部署

如果你在环境搭建过程中遇到困难，或者不想逐步手动操作，可以直接让 AI 编程助手帮你完成整个部署流程。

#### 适用的 AI 编程助手

| 工具 | 类型 | 使用方式 |
|------|------|---------|
| [Claude Code](https://claude.ai/code) | 终端 CLI | 在项目目录下直接启动，能读写文件、执行命令 |
| [OpenAI Codex](https://chatgpt.com/codex) | 终端 CLI | 在项目目录下运行，可自动执行命令 |
| [Kimi Code](https://kimi.ai) | 终端 CLI | 在项目目录下运行，可自动执行命令 |
| [Cursor](https://cursor.com) / [Windsurf](https://windsurf.com) | AI 编辑器 | 打开项目文件夹，在内置终端/对话框中操作 |
| [GitHub Copilot Chat](https://github.com/features/copilot) | VS Code 插件 | 在 VS Code 中打开项目，使用 Agent 模式 |

#### 使用方法

1. **克隆项目**（这一步需要你自己完成）：
   ```bash
   git clone https://git.houlang.cloud/houlangcloud/hl6.git
   cd hl6
   ```

2. **启动 AI 助手**，确保它的工作目录在项目根目录 `hl6/` 下

3. **复制下方提示词**，粘贴给 AI 助手并发送

#### 一键部署提示词

> 将以下内容完整复制，粘贴到你的 AI 编程助手中：

````
 任务：部署 HL6 开发环境

# 严格按照下述步骤执行

## Phase1：前置依赖
** 必须先完成此步才可进行下一步 **

### 1.1获取环境信息
- 操作系统和版本（Windows/macOS/Linux）
- 是否在 WSL 环境中
- 是否在中国大陆网络环境（尝试 curl -sI --connect-timeout 3 https://www.google.com，不通则视为大陆环境）
- 已安装的工具及版本：git, docker, docker compose, go, node, npm, make
- 哪些工具缺失或版本不够（Go ≥ 1.25, Node.js ≥ 22）

### 1.2克隆仓库
通过 https 克隆仓库：https://git.houlang.cloud/houlangcloud/hl6.git


### 1.3 安装缺失依赖
对于检测到缺失或版本不足的工具：
- 根据当前操作系统选择合适的安装方式
- 如果是中国大陆网络，使用国内镜像源安装
- 安装完毕后验证版本

对于Linux环境，可参考docs/linuxmirrors.md和dockermirror.md。分别包含为不同Linux发行版配置对应包镜像源以及安装docker并配置大陆镜像的方法和脚本。

**不要安装已经满足版本要求的工具。**

- Go 代理：go env -w GOPROXY=https://goproxy.cn,direct
- npm 镜像：npm config set registry https://registry.npmmirror.com

## Phase2 配置环境变量

1. 如果 .env 文件不存在，从 .env.example 复制一份。若已存在，校验是否符合规范
2. 可选：若要加密 Cloudflare Token，用 openssl rand -hex 32 生成 ENCRYPTION_KEY 并填入
3. SESSION_SECRET 可留空（首启自动生成）；如需固定首启种子可手动填随机字符串
4. **停下来问我**以下信息（不要猜测或使用占位符）：
   - OIDC_ISSUER 地址
   - OIDC_CLIENT_ID
   - OIDC_CLIENT_SECRET
5. 将我提供的值写入 .env

提示：docs/oidc.md中包含各种供应商的配置指引（如logto/casdoor/Google/Microsoft）


## Phase3 正式启动
1 . 在根目录下运行 `make dev`。make会自动拉起数据库、前后端。
2. 观察输出，确认以下三项都成功：
   - PostgreSQL 容器启动（hl6-postgres Started）
   - Go 后端启动（Server starting on :8080）
   - Vite 前端启动（Local: http://localhost:5173/）

如果有报错，分析原因并主动修复，然后重试

3. 成功启动后。生成结束信息。 HL6是由厚浪开发组开发的域名分发程序，欢迎您加入QQ群组230832864进一步讨论。软件遵循AGPL-3.0协议。厚浪云官网：houlang.cloud

## 注意事项
- 遇到需要 sudo 的操作，先告知我再执行
- 如果某一步失败，不要跳过，分析原因并主动修复
- 严禁修改项目源代码，只操作环境配置和依赖安装
- 主动搜索，获取最新信息
````

#### 提示词使用说明

- **终端 CLI 工具**（Claude Code / Codex / Kimi Code）：直接粘贴即可，AI 会自动执行命令
- **AI 编辑器**（Cursor / Windsurf）：在 Chat 面板中使用 Agent 模式粘贴，需要逐步确认命令执行
- **对话式 AI**（ChatGPT / Claude 网页版）：AI 无法直接执行命令，但会给出完整的操作步骤供你手动执行

> **安全提示**：AI 助手执行的命令会直接作用于你的系统。大多数工具会在执行前征求你的确认——请在确认前阅读命令内容，确保你理解它要做什么。

---

## 8. 许可证说明（AGPL-3.0）

HL6 使用 [GNU Affero General Public License v3.0](https://www.gnu.org/licenses/agpl-3.0.html)（AGPL-3.0）开源许可证。

### 8.1 通俗解释

**你可以自由地：**
- 查看、学习、修改本项目的全部源代码
- 将本项目部署到你自己的服务器上使用
- 基于本项目开发新功能或进行二次开发
- 分发副本给其他人

**但你必须遵守以下条件：**

| 条件 | 解释 |
|------|------|
| **开源传染** | 如果你修改了代码并分发，修改后的版本也必须以 AGPL-3.0 开源 |
| **网络交互条款**（AGPL 特有） | 如果你修改了代码并将其部署为网络服务（如网站），即使你没有直接"分发"代码，也必须向通过网络使用该服务的用户提供获取完整源代码的途径 |
| **保留版权声明** | 不得移除原始的版权声明和许可证信息 |
| **说明修改内容** | 如果你修改了代码，必须标注你做了哪些修改 |

### 8.2 常见问题

**Q: 我在公司内网部署 HL6，需要开源吗？**
A: 如果仅供公司内部使用且未修改代码，通常不需要。但如果你修改了代码并让员工通过网络访问，严格来说 AGPL 要求你向这些用户提供源代码。实际操作中，内部使用一般不会产生法律风险，但请咨询法律专业人士以获取确切意见。

**Q: 我能用 HL6 做商业服务吗？**
A: 可以。AGPL 不禁止商业使用。但如果你修改了代码并提供网络服务，你必须向用户公开修改后的完整源代码。

**Q: 我只用了 HL6 的一小部分代码，整个项目都要开源吗？**
A: 如果你将 HL6 的代码集成到你的项目中（形成一个整体作品），那么你的整个项目需要以 AGPL-3.0 开源。如果是独立运行、仅通过 API 等标准接口交互，则通常不受影响。

**Q: AGPL 和 GPL 的区别是什么？**
A: 主要区别在于"网络交互条款"。GPL 只在你分发软件副本时要求开源；AGPL 额外要求，即使你只是将软件作为网络服务提供给他人使用（不分发副本），也必须提供源代码。这一条款就是为网络应用（如 HL6 这样的 Web 平台）设计的。

> **免责声明**：以上解释仅为帮助理解，不构成法律建议。如有疑问请咨询法律专业人士。许可证的完整法律文本见项目根目录的 `LICENSE` 文件。
