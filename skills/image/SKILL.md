---
name: image
version: 0.1.28
description: 图片格式转换——PNG/JPG/WebP/GIF 互转、缩放、ICO 图标生成、Base64 编码。用户要转格式、改尺寸、做 favicon 时使用。【重要】必须先读 skill 原文获取正确命令格式，禁止瞎猜。
---

# 图片格式转换

## 规则

**用户要转图片/改尺寸/做 ICO/转 Base64 → 跑 cyeam image convert → 返回结果。**

```
用户: "把这张 PNG 转 JPG"            →  cyeam image convert input.png -f jpg
用户: "把这图缩小到宽 800"          →  cyeam image convert input.png -w 800 -f jpg
用户: "做一个 favicon.ico"          →  cyeam image convert input.png -f ico
用户: "转成 WebP"                   →  cyeam image convert input.png -f webp
用户: "转成 Base64"                 →  cyeam image convert input.png -f base64
```

## 命令

```
cyeam image convert <input> -f <format> [options]
```

参数：

| 参数 | 说明 |
|------|------|
| `-f, --format` | 目标格式（必填）：jpg / png / webp / gif / ico / base64 |
| `-o, --out` | 输出文件路径（未指定时用 `<原名>.<新扩展名>`）|
| `-q, --quality` | JPG 质量 1-100（默认 90，WebP 无损，此参数无效）|
| `-w, --width` | 目标宽度 px |
| `-H, --height` | 目标高度 px |
| `--keep-ratio` | 只给宽或高时按比例缩放（默认 true） |

## 支持的格式

- **输入**：PNG / JPG / GIF / WebP / SVG（栅格化）
- **输出**：PNG / JPG / GIF / WebP（无损） / ICO（多尺寸 16/32/48px favicon） / Base64（data URL）

## 输出

- `jpg/png/webp/gif/ico`：写文件，stdout 输出 `saved: <path>`
- `base64`：直接在 stdout 输出 data URL 文本（可配合 `--pretty` 干净输出）

## 注意事项

- 不需要登录
- SVG 输入会被栅格化为位图（256×256 默认，可通过宽高指定）
- ICO 输出自动包含 16/32/48 三个尺寸，浏览器按需要取
- WebP 输出为无损 VP8L（保留所有细节）
- 用 `--pretty` 去掉 JSON 信封，只看输出
