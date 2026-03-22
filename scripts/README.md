# Build & Packaging Guide / 构建与打包指南

This directory contains scripts for building, packaging, and deploying the Secbot Manager service.
本目录包含用于构建、打包和部署 Secbot Manager 服务的脚本。

## Scripts / 脚本说明

### 1. `build_release.sh`
The main script for creating a distributable release package.
用于创建可分发发布包的主脚本。

**Usage / 用法:**
```bash
./scripts/build_release.sh
```

**Workflow / 构建流程:**
1.  **Prepare Assets / 准备资源**: 
    *   Checks for desktop application binaries (DMGs) in the `build/` directory.
    *   Copies and renames them to standard names in `build/downloads/` (e.g., `ClawdSecbot-mac-arm64.dmg`).
    *   *Important*: Ensure the raw DMG files (e.g., `ClawdSecbot-0.1.0-arm64-Personal.dmg`) exist in `build/` before running.
    *   检查 `build/` 目录下的桌面应用安装包 (DMG)。
    *   将其复制并重命名为标准名称存入 `build/downloads/` (例如 `ClawdSecbot-mac-arm64.dmg`)。
    *   *注意*：运行前请确保原始 DMG 文件已存在于 `build/` 目录中。

2.  **Build Docker Images / 构建 Docker 镜像**:
    *   `secbot-platform`: Go backend service (based on `golang:1.25-alpine`).
    *   `secbot-website`: Nginx frontend. It embeds the static website code AND the `downloads` folder created in step 1.
    *   `secbot-platform`: Go 后端服务。
    *   `secbot-website`: Nginx 前端。包含静态网站代码以及第1步准备好的 `downloads` 下载目录。

3.  **Export & Package / 导出与打包**:
    *   Exports Docker images to `.tar` files.
    *   Packages images, `docker-compose.yaml`, and `run.sh` into `secbot-release.tar.gz`.
    *   导出 Docker 镜像为 `.tar` 文件。
    *   将镜像、编排文件和运行脚本打包为 `secbot-release.tar.gz`。

**Output / 输出产物:**
*   `build/secbot-release.tar.gz`

### 2. `run.sh`
The deployment script included in the final release package.
包含在最终发布包中的部署脚本。

**Usage / 用法:**
```bash
# Extract the package first / 先解压包
tar -xzf secbot-release.tar.gz
cd secbot-release

# Run deployment / 运行部署
./run.sh
```

**Function / 功能:**
*   Loads Docker images (`secbot-platform.tar`, `secbot-website.tar`).
*   Starts services using `docker-compose`.
*   加载 Docker 镜像。
*   使用 `docker-compose` 启动服务。

## Project Structure / 项目结构

*   `build/`: Build artifacts.
    *   `downloads/`: Staging area for downloadable assets.

## Nginx Configuration / Nginx 配置
The website image uses a custom `nginx.conf` to handle:
*   **Localization**: Serving `index_cn.html` or `index_en.html` on the root path `/` based on the `language` cookie.
*   **Caching**: Disabling cache for index files to ensure language switching works immediately.
*   **Proxy**: Forwarding `/version` API requests to the backend.

网站镜像使用自定义 `nginx.conf` 处理：
*   **国际化**：根据 `language` cookie 在根路径 `/` 返回中文或英文页面。
*   **缓存**：禁用首页缓存，确保语言切换立即生效。
*   **代理**：将 `/version` API 请求转发至后端。
