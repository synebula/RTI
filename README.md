# Remote Text Input

🌍 [English](#english) | 🇨🇳 [简体中文](#简体中文)

---

## <a name="english"></a>🇬🇧 English

### Introduction

**Remote Text Input** is a lightweight, Go-based application initially designed for Hyprland scenarios. Its primary goal is to **allow users to input text on their Linux desktop using their mobile phone's native keyboard** via a simple web interface.

This tool is highly beneficial when you don't have immediate access to a physical keyboard, or when you wish to bypass complex Linux Input Method Engine (IME) setups simply by using the fully functional typing experience on your mobile device.

### Core Features & How It Works

- **Instant Phone-to-Desktop Typing**: Scan the generated QR code with your phone to open a web-based text input portal. Text sent from the phone is instantly injected into your computer's actively focused window.
- **Terminal Compatibility (Terminal Mode)**: Includes a specific "Terminal Mode" toggle on the phone UI. Since Linux terminals usually require `Ctrl+Shift+V` to paste, enabling this mode intelligently switches the backend simulating standard `Ctrl+V` to `Ctrl+Shift+V` (using `wtype`) to prevent pasting gibberish into terminals.
- **One-Click Send and Enter**: Offers functionality to send just an `Enter` key stroke, or to "Send Text + Enter" combined, optimizing interactions for chat applications and command lines.
- **Single Binary Deployment**: The entire project, including the mobile web user interface (HTML/JS/CSS), is bundled into a single executable binary using Go `embed`, requiring zero external static setups to run.
- **Secure Access**: Utilizes a randomized URL token generated upon startup to ensure that unauthorized users on the same local network cannot connect or type into your machine.
- **Clipboard Restoration**: Although it relies on the clipboard (`wl-copy`) to transfer text temporarily, it makes a best-effort attempt to dynamically restore your original clipboard contents right after the paste is completed.

### How to Run

**1. Run directly from code:**
```bash
go run .
# Or run the main file specifically:
go run ./main.go
```

**2. Useful Runtime Flags:**
- `go run . --pair`: Starts the server and automatically opens a friendly Pairing Page in your desktop browser showing the QR code.
- `go run . --dry-run`: Tests the connection and UI flow without actually simulating key strokes or injecting text into windows.
- `go run . --log`: Enables verbose logging to output received text payloads and connections, useful for debugging.
- `go run . --debug`: Reuses a constant token (`remote-text-input-debug`) on restarts and forces logging on, perfect for rapid development.

**3. Build a standalone binary:**
```bash
go build -o remote-text-input .
./remote-text-input
```

**4. Build for Windows (Cross-compilation from Linux):**
```bash
CGO_ENABLED=0 GOOS=windows go build -o remote-text-input.exe .
```

### Known Limitations
- Currently relies on clipboard + paste (`wl-copy` & shortcut simulation) rather than a native Wayland IME protocol.
- Terminal Mode depends on your GUI terminal specifically understanding `Ctrl+Shift+V` and having `wtype` available in the system.
- Depends on the `Hyprland` default `sendshortcut` dispatcher format for non-terminal key operations.
- The web page UI assumes the user is actively managing focus on the desktop. The server will just type into whatever window is currently active.

---

## <a name="简体中文"></a>🇨🇳 简体中文

### 项目简介

**Remote Text Input** 是一个轻量级的、使用 Go 语言编写的远程文本输入工具（最初为 Hyprland 桌面环境定制）。它的核心作用是**允许用户通过手机的原生输入法向 Linux 桌面正在交互的窗口中直接输入文本**。

当你不方便使用实体键盘，或者在 Linux 桌面遇到原生输入法（IME）配置繁琐、导致无法正常打字等问题时，你可以直接使用手机扫描控制台或本地展示生成的二维码。网页打开后，你就可以像平时发消息聊天一样，用手机输入各种语言文本并立刻投射至你的电脑屏幕。

### 核心功能与工作原理

- **即扫即输的跨端打字**：程序启动后会展示配对二维码和链接。使用手机访问该局域网网页，通过手机输入并点击发送后，电脑端将拦截此文本，并通过 `wl-copy` 写入剪贴板，再通过系统底层自动模拟“粘贴快捷键”将其推入当前焦点窗口。
- **终端免缝兼容（Terminal Mode）**：原生的 Linux 终端应用常常需要通过 `Ctrl+Shift+V` 才可以粘贴文本。为此网页端内建支持了“终端模式”开关，开启后程序将智能使用 `wtype` 触发该专属按键行为，解决终端下直接粘贴快捷键冲突的问题。
- **一键回车与连贯操作**：除了基本文本发送，还支持单独发送 `Enter` 回车键，或者“发送文本并直接回车”，这让在命令行敲击命令或者聊天框里的发送动作一气呵成。
- **极简的单二进制交付**：所有前端的 Web UI 模板（基于 HTML/JS）均被通过 Go 的 `embed` 特性完美打包在二进制内部。无须部署 Nginx 等服务环境，只需要一次 `go build` 就能生成即开即用的单文件服务。
- **局域网安全保护机制**：每次拉起服务时默认会利用算法随机生成无规律的 URL Token。只有包含此令牌的请求会被放行，有效防止同层网络内的他人连接和恶意注入。
- **友好的剪贴板控制**：为实现文本中转，程序借用了操作系统的剪贴板通道，但其内部引擎会在输入完成后尽可能地（Best effort）帮你立即恢复你原有的剪贴板内容物。

### 运行参数指南

**1. 基础启动：**
```bash
go run .
# 也可以直接指出入口文件：
go run ./main.go
```

**2. 进阶运行参数：**
- `go run . --pair`: 启动并在桌面的默认浏览器里弹出一个美观的配对页（展示大尺寸二维码和访问链接）。
- `go run . --dry-run`: 演练模式（Dry Run），服务端会响应所有的操作并在日志里打印准备执行什么类型的粘贴行为，但不会真正触发按键注入当前窗口。
- `go run . --log`: 强制显示详尽的日志，比如手机端推送的文本负载报文内容，方便环境内排查。
- `go run . --debug`: 固定使用调试专属 Token (`remote-text-input-debug`)，并自动关联打开详尽日志。极大方便前端和 Go 服务热调试重启。

**3. 构建单文件可执行产物：**
```bash
go build -o remote-text-input .
./remote-text-input
```

**4. 交叉编译 Windows 版：**
```bash
CGO_ENABLED=0 GOOS=windows go build -o remote-text-input.exe .
```

### 已知限制
- 目前仍是 MVP 验证阶段，核心机制依赖的是“借用剪贴板并模拟按键发送粘贴”，而不是 Wayland 桌面下标准的底层 IME 输入法注入通道。
- 手机端只负责文本内容的发出，不会辅助操作桌面的焦点。目标程序窗口需要在 Linux 侧提前保持聚焦 (active) 状态才能收到输入。
- 特性重度依赖相关依赖项，例如你需要确保你的 Linux 环境里具备 `wtype` ，并且 `Hyprland` 已经正确配置其原生的 `sendshortcut` 控制。
