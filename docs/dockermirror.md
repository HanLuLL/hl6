`bash <(curl -sSL https://linuxmirrors.cn/docker.sh) --help `

|          名称           |                  含义                   |            选项值            |
| :---------------------: | :-------------------------------------: | :--------------------------: |
|       `--source`        |    指定 `Docker CE` 源地址(域名或IP)    |            `地址`            |
|   `--source-registry`   |  指定 `Docker` 镜像仓库地址(域名或IP)   | `地址（多个用英文逗号分割）` |
|       `--branch`        |    指定 `Docker CE` 软件源仓库(路径)    |   `仓库名（详见下方文档）`   |
|   `--branch-version`    |     指定 `Docker CE` 软件源仓库版本     |   `版本号（详见下方文档）`   |
| `--designated-version`  |      指定 `Docker Engine` 安装版本      |   `版本号（详见下方文档）`   |
|      `--codename`       |   指定 `Debian` 系操作系统的版本代号    |          `代号名称`          |
|      `--protocol`       |     指定 `Docker CE` 源的 Web 协议      |      `http` 或 `https`       |
| `--use-intranet-source` | 是否优先使用内网 `Docker CE` 软件源地址 |      `true` 或 `false`       |
|   `--install-latest`    |   是否安装最新版本的 `Docker Engine`    |      `true` 或 `false`       |
|   `--close-firewall`    |             是否关闭防火墙              |      `true` 或 `false`       |
|    `--clean-screen`     |    是否在运行前清除屏幕上的所有内容     |      `true` 或 `false`       |
|        `--lang`         |           指定脚本输出的语言            |   `语言ID（详见下方文档）`   |
|    `--only-registry`    |           仅更换镜像仓库模式            |              无              |
| `--ignore-backup-tips`  |    忽略覆盖备份提示（即不覆盖备份）     |              无              |
|      `--pure-mode`      |         纯净模式，精简打印内容          |              无              |
|        `--help`         |              查看帮助菜单               |                              |

可用 Docker 镜像：docker.hlmirror.com