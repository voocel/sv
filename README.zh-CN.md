<p align="center" style="color: red">
    <h1 align="center">SV</h1>
    <p align="center">Switch golang version</p>
</p>

**SV** 一个轻量级、漂亮的 Go 版本管理器

[English](./README.md) | **中文**

![Example](./sv1.gif)

## 🏆 简介
方便你轻松安装和切换不同的 Go 版本

## ⬇️️ 安装

### Linux / macOS

```bash
curl -sL https://raw.githubusercontent.com/voocel/sv/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/voocel/sv/main/install.ps1 | iex
```

> 安装完成后，请打开**新的终端窗口**使配置生效。

## 🔥 功能
- 列出本地或远程所有版本
- 安装指定版本
- 卸载指定版本
- 快速切换本地版本
- 漂亮的下载进度条
- 支持断点续传
- 清理旧版本
- 检查版本更新

## 🌲 使用方法

**列出并选择要安装的版本**
```bash
sv list           # 本地版本
sv list -r        # 远程版本
```

**安装指定版本**
```bash
sv install 1.23.4
sv install --latest   # 安装最新稳定版
```

**切换到指定版本**
```bash
sv use 1.23.4
```

**卸载指定版本**
```bash
sv uninstall 1.18.1
```

**其他命令**
```bash
sv current          # 显示当前使用的版本
sv latest           # 显示最新可用版本
sv outdated         # 检查已安装版本是否过时
sv where 1.23.4     # 显示安装路径
sv prune            # 清理旧版本，保留最近的
sv self upgrade     # 升级 sv 本身
sv self uninstall   # 卸载 sv 及所有 Go 版本
```

## 💡 许可证

Copyright © 2016–2025

Licensed under [Apache License 2.0](/LICENSE)

## 🙋 贡献

欢迎参与贡献！
