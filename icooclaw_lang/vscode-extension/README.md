# iclang VS Code Extension

为 `icooclaw script language` 提供基础编辑器支持：

- `.is` 文件识别
- 关键字高亮
- 字符串、注释、数字高亮
- 内置函数、标准库成员访问高亮
- 基础括号和注释配置
- 常用代码片段

## 本地调试

1. 在 VS Code 中打开当前目录
2. 打开 `icooclaw_lang/vscode-extension/`
3. 按 `F5` 启动 Extension Development Host
4. 在新窗口中打开 `.is` 文件验证高亮

## 打包

如果本机已安装 Node.js：

```bash
cd icooclaw_lang/vscode-extension
npm install
npm run package:out
```
