# SniShaper

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey?style=flat-square)]()
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)]()

通过透传、本地中间人重写或 TLS 分片绕过 SNI 阻断的桌面代理工具，支持普通域前置和 ECH fronting。

## 核心特性

- **四模式代理**
  - `transparent`：透明透传（TCP 隧道，不解密 TLS，完成自定义 host）
  - `mitm`：中间人模式（本地 CA 解密，修改 sni 绕过阻断）
  - `tls-rf`：TLS 分片模式（不解密 TLS，在 ClientHello 阶段做分片发送）
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
浏览器 → SniShaper(127.0.0.1:8080) → 规则匹配 → [模式选择: transparent/mitm/tls-rf] → 上游握手 (ECH/uTLS/TLS Fragment) → 目标直连
```

## 快速开始

### 1. 启动
运行 `snishaper.exe`。默认监听端口为 `127.0.0.1:8080`（可在设置中修改）。

## 构建

推荐使用仓库内脚本而不是直接执行 `wails build`：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1
```

这个脚本会在构建完成后自动把运行时配置同步到 `build/bin/rules/config.json`。

### 2. 安装证书（MITM 模式必需）
点击界面「证书管理」按钮，安装生成的根证书到「受信任的根证书颁发机构」，并重启浏览器。

### 3. 配置加速
在 **Cloudflare ECH** 页面输入想要加速的域名，点击添加即可一键生成配置。

### 4. 启用代理
点击主界面的「启动代理」并开启「系统代理」即可。

### 5. 选择合适模式
- `mitm`：兼容性最好，适合需要完整 HTTPS 代理能力的场景
- `transparent`：不解密 TLS，适合简单隧道转发
- `tls-rf`：适合不想安装证书、但目标链路对 SNI/ClientHello 敏感的场景
- `server`：适合需要借助远端中转节点的场景

## 配置字段说明

| 字段 | 说明 |
|------|------|
| `domains` | 域名匹配列表 |
| `website` | 网站分组名（用于 UI 聚合展示） |
| `mode` | `transparent`、`mitm`、`tls-rf` 或 `server` |
| `upstream` | 上游地址（可指定 IP:443 或留空由程序自动解析） |
| `sni_policy` | SNI 处理策略 |
| `utls_policy` | 指纹伪装策略 (`on` / `off` / `auto`) |
| `ech_enabled` | 是否开启 ECH 加密（绕过封锁的关键） |
| `use_cf_pool` | 是否启用优选 IP 池平衡负载与稳定性 |

## TLS 分片说明

`tls-rf` 模式不会像 MITM 那样终止客户端 TLS，也不会像透明模式那样完全原样透传。它会在转发到上游时对 TLS ClientHello 做分片发送，用来降低部分链路对单包 SNI 特征的识别成功率。

适用场景：

- 不希望安装本地根证书
- 目标站点对完整 MITM 很敏感
- 透明模式仍然容易被基于 ClientHello 的检测命中

注意事项：

- 这不是通用绕过手段，效果取决于目标网络环境
- 某些站点或中间设备可能对分片握手更敏感，必要时应回退到 `mitm` 或 `transparent`
- `tls-rf` 主要影响 CONNECT/TLS 链路，普通 HTTP 明文请求不会因此获得额外绕过能力

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

4. 在客户端配置中填写 Tunnel给的域名和鉴权密码



## 常见问题

- **证书错误**：请确认证书已导入「受信任的根证书」分类，并务必重启浏览器
- **访问速度慢**：建议在「优选 IP 池」中添加更多当前环境下延迟较低的 Cloudflare 任播 IP

## 致谢

本项目在开发过程中参考并受益于以下优秀开源项目：

- [SNIBypassGUI](https://github.com/racpast/SNIBypassGUI)
- [DoH-ECH-Demo](https://github.com/0xCaner/DoH-ECH-Demo)
- [lumine](https://github.com/moi-si/lumine)

## 许可

[MIT License](LICENSE)
