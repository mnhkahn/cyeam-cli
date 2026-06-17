---
name: live-broadcast
version: 0.1.13
description: 直播预报——查询 NBA / 世界杯 / 中国男女足比赛的赛程时间和转播源（CCTV5、央视频、腾讯体育、咪咕视频等）。想看球但不知道哪播、几点播的时候用。
---

# 直播预报

## 概述

查询 NBA、世界杯、中国男足/女足国家队的赛程，返回**比赛时间 + 对阵 + 阶段 + 转播源**。不依赖登录，不调用后端服务。

```bash
cyeam tv list                        # 未来 3 天全部比赛
cyeam tv list --league nba           # 只看 NBA
cyeam tv list --league worldcup      # 只看世界杯
cyeam tv list --team 湖人            # 只看湖人
cyeam tv list --source CCTV5         # 只看 CCTV5 播的
cyeam tv today                       # 今天
cyeam tv next                        # 下一场
```

## 安装

二进制安装后即可使用，无需登录：

| 平台 | 安装命令 |
|------|----------|
| macOS Apple Silicon | `curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_arm64.tar.gz \| tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/` |
| macOS Intel | `curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Darwin_x86_64.tar.gz \| tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/` |
| Linux amd64 | `curl -L https://github.com/mnhkahn/cyeam-cli/releases/latest/download/cyeam_Linux_x86_64.tar.gz \| tar xz && chmod +x cyeam && sudo mv cyeam /usr/local/bin/` |

检查是否已安装：`which cyeam && cyeam version`

## Flags

| Flag | 简写 | 默认 | 说明 |
|------|------|------|------|
| `--league` | `-l` | 全部 | 联赛过滤：`nba`、`worldcup`、`cn-football`（可重复/逗号分隔） |
| `--team` | `-t` | — | 球队过滤，支持中文/英文/缩写，如 `湖人`、`LAL`、`阿根廷` |
| `--source` | — | — | 转播源过滤，如 `CCTV5`、`腾讯` |
| `--days` | `-d` | `3` | 未来 N 天（最大 14） |
| `--from` | — | — | 起始日期 `YYYY-MM-DD` |
| `--to` | — | — | 结束日期 |
| `--tz` | — | `Asia/Shanghai` | 显示时区 |
| `--json` | — | — | 输出 JSON |
| `--include-finished` | — | — | 包含已结束的比赛 |
| `--no-color` | — | — | 关闭彩色输出 |

## 使用示例

```bash
# 联赛过滤
cyeam tv list --league nba --days 7
cyeam tv list --league worldcup,cn-football
cyeam tv list --league nba --from 2026-06-15 --to 2026-06-20

# 球队和转播源过滤
cyeam tv list --team 湖人
cyeam tv list --source CCTV5

# JSON 输出（适合进一步处理）
cyeam tv next --json
cyeam tv list --json --league nba --days 7
```

## 输出

表格模式（默认）：

```
开始时间             联赛       比赛                 阶段            转播源
─────────────────   ────────   ─────────────────   ────────────   ──────────────────────────
06-13 周六 03:00    世界杯     加拿大 vs Bosnia      -              CCTV5、央视频、咪咕视频
06-14 周日 08:30    NBA 总决赛  马刺 vs 尼克斯       Finals G5      CCTV5、腾讯体育、咪咕视频
```

转播源在支持的终端（iTerm2、kitty、WezTerm）可 cmd+click 打开。

## 联赛覆盖

| league | 覆盖 |
|--------|------|
| `nba` | 常规赛、附加赛、季后赛、总决赛、季前赛、全明星、杯赛 |
| `worldcup` | 男足世界杯 + 女足世界杯 |
| `cn-football` | 中国男足/女足全部国际比赛（友谊赛、世预赛、亚洲杯、东亚杯、奥运会） |

## 注意事项

- 国足比赛仅在 FIFA 国际比赛日窗口期有数据，平时无比赛正常
- NBA 赛程 JSON 约 10MB，首次 5–15s，已缓存
- 单数据源不可达时仅该联赛降级提示，不影响其他联赛
- 数据源：NBA 来自 `cdn.nba.com`，世界杯/国足来自 ESPN site API

## 不做的事

- 不抓直播流、不解 m3u8/flv
- 不绕过会员或地区限制
- 不推送实时比分或赛后集锦
- 不覆盖 NBA / 世界杯 / 中国男女足以外的赛事