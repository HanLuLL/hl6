`bash <(curl -sSL https://linuxmirrors.cn/main.sh) --help `

| 名称                         | 含义                                                         |          选项值          |
| :--------------------------- | :----------------------------------------------------------- | :----------------------: |
| `--abroad`                   | 使用境外以及海外软件源                                       |            无            |
| `--edu`                      | 使用中国大陆教育网软件源                                     |            无            |
| `--source`                   | 指定软件源地址（域名或IP）                                   |          `地址`          |
| `--source-epel`              | 指定 EPEL 附加软件包仓库的软件源地址（域名或IP）             |          `地址`          |
| `--source-security`          | 指定 Debian / Ubuntu 系统 security 仓库的软件源地址（域名或IP） |          `地址`          |
| `--source-vault`             | 指定 CentOS / AlmaLinux 系统 vault 仓库的软件源地址（域名或IP） |          `地址`          |
| `--source-portage`           | 指定 Gentoo 系统 portage 仓库的软件源地址（域名或IP）        |          `地址`          |
| `--source-base-system`       | 指定 Linux Mint / Raspberry Pi OS 底层系统的软件源地址（域名或IP） |          `地址`          |
| `--branch`                   | 指定软件源仓库（路径）                                       |         `仓库名`         |
| `--branch-epel`              | 指定 EPEL 附加软件包仓库的软件源仓库（路径）                 |         `仓库名`         |
| `--branch-security`          | 指定 Debian 系统 security 仓库的软件源仓库（路径）           |         `仓库名`         |
| `--branch-vault`             | 指定 CentOS / AlmaLinux 系统 vault 仓库的软件源仓库（路径）  |         `仓库名`         |
| `--branch-portage`           | 指定 Gentoo 系统 portage 仓库的软件源仓库（路径）            |         `仓库名`         |
| `--branch-base-system`       | 指定 Linux Mint / Raspberry Pi OS 底层系统的软件源仓库（路径） |         `仓库名`         |
| `--codename`                 | 指定 Debian 系 / openKylin 操作系统的版本代号                |        `代号名称`        |
| `--protocol`                 | 指定 Web 协议                                                |    `http` 或 `https`     |
| `--use-intranet-source`      | 是否优先使用内网软件源地址                                   |    `true` 或 `false`     |
| `--use-official-source`      | 是否使用目标操作系统的官方软件源                             |    `true` 或 `false`     |
| `--use-official-source-epel` | 是否使用 EPEL 附加软件包的官方软件源                         |    `true` 或 `false`     |
| `--install-epel`             | 是否安装 EPEL 附加软件包                                     |    `true` 或 `false`     |
| `--backup`                   | 是否备份原有软件源                                           |    `true` 或 `false`     |
| `--upgrade-software`         | 是否更新软件包                                               |    `true` 或 `false`     |
| `--clean-cache`              | 是否在更新软件包后清理下载缓存                               |    `true` 或 `false`     |
| `--clean-screen`             | 是否在运行前清除屏幕上的所有内容                             |    `true` 或 `false`     |
| `--lang`                     | 指定脚本输出的语言                                           | `语言ID（详见下方文档）` |
| `--only-epel`                | 仅更换 EPEL 软件源模式                                       |            无            |
| `--ignore-backup-tips`       | 忽略覆盖备份提示（即不覆盖备份）                             |            无            |
| `--print-diff`               | 是否打印源文件修改前后差异                                   |            无            |
| `--pure-mode`                | 纯净模式，精简打印内容                                       |            无            |
| `--help`                     | 查看帮助菜单                                                 |            无            |

** 在 https://all.hlmirror.com/{url} 填入需要代理的 url，即可在大陆地区访问该网页**

eg. https://all.hlmirror.com/https://google.com/