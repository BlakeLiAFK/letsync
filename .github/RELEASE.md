# 发布流程

本项目使用 GitHub Actions 自动化发布流程。

## 如何发布新版本

1. **确保所有更改已提交并推送到主分支**

2. **创建版本标签**

```bash
# 创建标签 (例如 v1.0.0)
git tag -a v1.0.0 -m "Release v1.0.0"

# 推送标签到远程仓库
git push origin v1.0.0
```

3. **GitHub Actions 自动构建**

推送标签后,GitHub Actions 会自动:
- 构建前端 (Vue + Vite)
- 为多个平台构建二进制文件:
  - `letsyncd-linux-amd64` - Linux AMD64
  - `letsyncd-linux-arm64` - Linux ARM64
  - `letsyncd-linux-armv7` - Linux ARMv7
  - `letsyncd-darwin-amd64` - macOS Intel
  - `letsyncd-darwin-arm64` - macOS Apple Silicon
  - `letsyncd-windows-amd64.exe` - Windows AMD64
- 生成 SHA256 校验和文件
- 创建 GitHub Release 并上传所有构建产物

4. **查看发布**

访问项目的 Releases 页面查看自动生成的发布:
https://github.com/你的用户名/letsync/releases

## 版本号规范

遵循语义化版本规范 (Semantic Versioning):

- **主版本号** (MAJOR): 不兼容的 API 修改
- **次版本号** (MINOR): 向下兼容的功能性新增
- **修订号** (PATCH): 向下兼容的问题修正

示例:
- `v1.0.0` - 初始发布
- `v1.1.0` - 新增功能
- `v1.1.1` - 修复问题

## 本地测试版本号注入

```bash
# 测试版本号注入
go build -ldflags="-X main.Version=1.0.0-test" -o letsyncd ./cmd/letsyncd
./letsyncd -v
# 输出: Letsync v1.0.0-test
```

## 注意事项

- 只有推送以 `v` 开头的标签才会触发发布流程
- 删除标签后重新创建相同标签可能需要使用 `--force` 选项
- 构建产物会自动嵌入前端资源,无需单独部署前端
