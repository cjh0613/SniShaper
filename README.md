# SniShaper

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey?style=flat-square)]()
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)]()

通过透传或本地中间人重写绕过 SNI 阻断的桌面代理工具，支持普通域前置和 ECH fronting。

## 核心特性

- **三模式代理**
  - `transparent`：透明透传（TCP 隧道，不解密 TLS，完成自定义 host）
  - `mitm`：中间人模式（本地 CA 解密，修改 sni 绕过阻断）
  - `server`：服务端模式（连接上游动态反代服务器，无感中转）

- **Cloudflare ECH 一键加速**
  - **动态 ECH**：通过内置 DoH 处理，动态获取 ECH 配置，避免明文 SNI 暴露导致的阻断
  - **全球优选 IP 池**：智能优选边缘节点，极大提升访问稳定性

- **协议深度控制**
  - `sni_policy`：自定义 SNI 行为（fake / original / upstream / none）
  - `utls_policy`：内置指纹伪装，有效对抗针对性探查
  - `alpn_policy`：灵活控制 HTTP 协议版本选择

## 工作原理

```
浏览器 → SniShaper(127.0.0.1:8080) → 规则匹配 → [模式选择: transparent/mitm] → 上游握手 (ECH/uTLS) → 目标直连
```

## 快速开始

### 1. 启动
运行 `snishaper.exe`。默认监听端口为 `127.0.0.1:8080`（可在设置中修改）。

### 2. 安装证书（MITM 模式必需）
点击界面「证书管理」按钮，安装生成的根证书到「受信任的根证书颁发机构」，并重启浏览器。

### 3. 配置加速
在 **Cloudflare ECH** 页面输入想要加速的域名，点击添加即可一键生成配置。

### 4. 启用代理
点击主界面的「启动代理」并开启「系统代理」即可。

## 配置字段说明

| 字段 | 说明 |
|------|------|
| `domains` | 域名匹配列表 |
| `website` | 网站分组名（用于 UI 聚合展示） |
| `mode` | `transparent` 或 `mitm` |
| `upstream` | 上游地址（可指定 IP:443 或留空由程序自动解析） |
| `sni_policy` | SNI 处理策略 |
| `utls_policy` | 指纹伪装策略 (`on` / `off` / `auto`) |
| `ech_enabled` | 是否开启 ECH 加密（绕过封锁的关键） |
| `use_cf_pool` | 是否启用优选 IP 池平衡负载与稳定性 |

## 服务端部署

SniShaper 支持两种服务端部署方式：

### 方式一：Cloudflare Worker（免费）

```
客户端 → Worker → 目标网站
```

**部署步骤：**

1. 登录 [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. 创建一个新的 Worker（hello world 模板）
3. 将 `sni-server/worker.js` 的内容复制到 Worker 编辑器
4. 在 Worker 设置中添加环境变量 `AUTH_SECRET`，设置密码
5. 部署 Worker，获取 Worker 域名（如 `xxx.workers.dev`）
6. 在客户端 Server 节点设置中填写域名和鉴权密码

### 方式二：VPS 部署（sni-server）

```
客户端 → Cloudflare Tunnel → VPS (sni-server) → 目标网站
```

**部署步骤：**

1. 准备一台 VPS（任何支持 Go 的 Linux 服务器）
2. 运行一键部署脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/coolapijust/Shaper-Next/main/server/install.sh -o /tmp/sni-server-install.sh
sudo bash /tmp/sni-server-install.sh
```

3. 配置域名解析。推荐使用 Cloudflare Tunnel：
```bash
bash <(curl -sSL https://github.com/sky22333/shell/raw/main/dev/cf-tunnel.sh)
```

4. 在客户端配置中填写 VPS 地址和鉴权密码

### 对比

| 特性 | Worker | VPS (sni-server) |
|------|--------|------------------|
| 成本 | 免费 | 需要 VPS |
| 维护 | 无需 | 需要管理 |
| 出口 IP | Cloudflare IP | VPS IP |
| 带宽限制 | 有 | 取决于 VPS |
| 定制化 | 有限 | 完全控制 |

## 常见问题

- **证书错误**：请确认证书已导入「受信任的根证书」分类，并务必重启浏览器
- **访问速度慢**：建议在「优选 IP 池」中添加更多当前环境下延迟较低的 Cloudflare 任播 IP

## 致谢

本项目在开发过程中参考并受益于以下优秀开源项目：

- [SNIBypassGUI](https://github.com/racpast/SNIBypassGUI)
- [DoH-ECH-Demo](https://github.com/0xCaner/DoH-ECH-Demo)

## 许可

[MIT License](LICENSE)
