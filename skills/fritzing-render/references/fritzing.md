# 渲染与草图格式

上游资料见 [fritzing-render](https://github.com/LeonFedotov/fritzing-render) 和 [网页工具](https://leonfedotov.github.io/fritzing-render/)。

## 渲染已有草图

在草图所在目录运行。用实际文件名替换示例中的 `wiring.fzz`。

```sh
fritzing-render render wiring.fzz --view breadboard -o wiring.svg --png wiring.png
```

或使用上游提供的 Docker 镜像。

```sh
docker run --rm -v "$PWD:/work" ghcr.io/leonfedotov/fritzing-render \
  render wiring.fzz --view breadboard -o wiring.svg --png wiring.png
```

需要原理图时用 `--view schematic`，并换一个输出文件名。仅要 SVG 时省略 `--png`。在网页的 **Sketches → Your sketch** 区域选择文件或拖进去，切换 **Breadboard** / **Schematic**，再下载 SVG 或 PNG。图面若有后来添加的标题、图例或备注，更新展示图时一起核对。

## 建立网页可编辑的 JSON 草图

用 `fritzing-render search` 查元件，`fritzing-render part "<元件引用>"` 查端子名及坐标。也可在网页的 **Parts** 查看。优先按实际端子名或连接器 ID 连线。

```json
{
  "parts": [
    {"id": "controller", "generic": {"title": "Controller", "pins": ["GND", "DATA"]}, "x": 0, "y": 0},
    {"id": "module", "generic": {"title": "Module", "pins": ["GND", "DIN"]}, "x": 130, "y": 0}
  ],
  "wires": [
    {"from": "controller.DATA", "to": "module.DIN", "color": "green", "via": [[92, 85]]},
    {"from": "controller.GND", "to": "module.GND", "color": "black", "via": [[85, 105]]}
  ],
  "margin": 18
}
```

上例的 `generic` 是带端子名的示意块。使用库元件时，改用 `part` 指定搜索所得引用。坐标以每英寸 90 个单位计，`via` 是导线折点。网页可先导入自定义 `.fzpz`，再在 **Editor** 排布、连线并下载 JSON。分享时将这些自定义元件与 JSON 一起保存。

## 维护含自定义元件的 `.fzz`

`.fzz` 是 ZIP，包含 `.fz` 草图和随图打包的自定义元件。需要直接处理归档时，先在副本中保留已有草图、元件定义与 SVG。包内自定义文件常见命名为 `part.<name>.fzp`、`svg.breadboard.<name>.svg` 和 `svg.schematic.<name>.svg`。核对 `.fzp` 的模块 ID、视图和连接器引用，再核对 `.fz` 中导线端点的模块实例、`connectorId` 与各视图位置。

现有文件可参考上游的 [最小 `.fz` 示例](https://github.com/LeonFedotov/fritzing-render/blob/main/examples/blink.fz)。重新打包后用 `unzip -t` 检查归档，从新包渲染需要的视图，并核对连接。仅有 SVG 时可以编辑外观；需要继续维护元件和电气连接时，应使用原草图，或根据可核实的接线重建草图。
