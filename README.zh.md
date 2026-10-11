# Feishu CLI Next

[English](README.md) | 中文

面向通过终端和文件工作的 AI Agent 的飞书 / Lark CLI，基于官方 `larksuite/cli` 开发，保留原有命令、认证和扩展接口，并提供命令发现、紧凑 JSON、文件结果、类型化数据传递和变更凭据。

这是独立的开发版本，完整线上功能验证和正式发布验收尚未完成。公开 npm 包、二进制发布和自动更新渠道尚未配置；Windows 执行验证仍待完成。源码构建和本地安装可用，详见[分发说明](docs/distribution.md)。

## 构建与安装

需要 **Go 1.26.9+** 和 `make`；首次构建可能下载 Go 依赖。产品使用源码中已提交的 API 目录，构建无需 Python 或在线获取元数据。清单、打包、评估脚本及本地前缀安装工具需要 Python 3。

```sh
git clone https://github.com/Zhujingxi/feishu-cli-next.git
cd feishu-cli-next
make build
./feishu-cli-next --version
```

将二进制安装到个人目录：

```sh
make install PREFIX="$HOME/.local"
```

将 `$HOME/.local/bin` 加入 `PATH`。更新和卸载使用相同前缀的 `make update-install`、`make uninstall`；这些操作会校验本地安装归属，并拒绝覆盖无关或被修改的文件。

`@larksuite/cli` 是上游官方产品；安装该包不会安装 Feishu CLI Next。本项目当前仅提供源码及[本地归档和 npm 包安装流程](docs/distribution.md)。

## 配置和登录

先检查现有账号配置：

```sh
./feishu-cli-next auth status
```

首次使用时，配置飞书 / Lark 应用并完成授权：

```sh
./feishu-cli-next config init --new
./feishu-cli-next auth login --recommend
```

认证沿用上游 profile、凭证提供器、品牌选择及 `LARKSUITE_CLI_*` 环境变量。用户和机器人身份权限不同，详见[上游认证说明](README.upstream.md#authentication)。

## Agent 入口与命令

在已有项目中安装一个入口 Skill：

```sh
./feishu-cli-next guide install --project /path/to/your/project
```

入口安装在项目的 `.agents/skills/`；通过相同项目路径执行 `guide update` 或 `guide uninstall` 进行管理。它不会修改已有上游全局 Skills、账号或配置。

```sh
./feishu-cli-next contact --help
./feishu-cli-next discover 'contact open_id' --flags --limit 3
./feishu-cli-next schema drive files list
./feishu-cli-next contact +search-user --query 'Alex' --as user
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --as user
```

把示例占位符替换为自己的资源 ID。`+` 开头的快捷命令提供工作流，生成的服务命令对应已收录 API，`api` 支持直接调用 OpenAPI。命令存在不代表当前账号拥有所需权限。

默认结果为紧凑 JSON；`--result-detail` 提供缩进，支持的读命令可用 `--save-result PATH` 保存完整结果而不覆盖已有文件。检查结果中的完整性和变更状态以及进程退出码。超时后的变更可能已生效，应通过 `receipts list`、`receipts inspect ID` 核对凭据后再决定是否重试。

文档读取、局部文本编辑、Sheets / Base 文件工作流及已知并发限制见[英文使用说明](README.md#usage)和[入口 Skill](agent-skills/feishu-cli-next/SKILL.md)。继承的领域指南仍可通过 `skills list/read` 按需读取；其中的 `lark-cli` 示例在本分发中使用 `feishu-cli-next` 执行。

## 验证状态与贡献

[实现状态](docs/IMPLEMENTATION.md)记录离线测试、限定范围的线上验证及尚未完成的验收项。[评估报告](evaluation/evidence/full-matrix-20260911/REPORT.md)保留失败尝试、缺失用量及各项限制；有限样本不能证明所有任务的可靠性或统一 token 节省。

代码沿用 [MIT 许可证](LICENSE)和原有版权声明，Go 模块路径保留为 `github.com/larksuite/cli` 以兼容源码及扩展。开发规则见 [AGENTS.md](AGENTS.md)，输出契约见[输出说明](docs/contracts/output.md)，漏洞报告方式见 [SECURITY.md](SECURITY.md)。
