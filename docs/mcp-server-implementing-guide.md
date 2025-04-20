# Model Context Protocol (MCP) Server実装ガイド

## MCP概要

Model Context Protocol（MCP）は、AI言語モデル（LLM）と外部データソースやツールを標準化された方法で接続するためのオープンプロトコルです。LLMアプリケーションと外部ツールの間のプラグインシステムとして機能し、データソースへのシームレスなアクセスを提供します。

## 基本アーキテクチャ

MCPはクライアント-サーバーモデルに基づいています：

- **MCPサーバー**: データソースやツールへのアクセスを提供する軽量プログラム
- **MCPホスト/クライアント**: Claude DesktopなどのLLMアプリケーションで、MCPサーバーに接続して機能を利用

## MCPサーバーの主な機能

MCPサーバーは以下の3つの主要な機能タイプを提供できます：

1. **ツール（Tools）**: LLMから呼び出し可能な関数（ユーザー承認付き）
2. **リソース（Resources）**: クライアントが読み取り可能なファイル形式のデータ（APIレスポンスやファイル内容など）
3. **プロンプト（Prompts）**: 特定のタスク実行に役立つ定型テンプレート

## 通信プロトコル

MCPサーバーは以下の通信方式をサポートしています：

- **標準入出力（stdio）**: ローカル開発に適したシンプルな方式
- **Server-Sent Events (SSE)**: より柔軟性の高い分散チーム向けの方式
- **WebSockets**: リアルタイム双方向通信向け

## 実装に必要な要素

### 1. 基本構造

MCPサーバーの実装には以下の要素が必要です：

- サーバー初期化とトランスポート設定
- ツール/リソース/プロンプトの定義
- リクエストハンドラーの実装
- メインサーバー実行関数

### 2. サーバーの設定

サーバーの設定には以下の情報が含まれます：

```json
{
  "mcpServers": {
    "myserver": {
      "command": "実行コマンド",
      "args": ["引数1", "引数2"],
      "env": {
        "環境変数名": "値"
      }
    }
  }
}
```

## 言語別実装方法の概要

MCPサーバーは様々なプログラミング言語で実装可能です：

### Python

```python
# 必要なライブラリをインストール
# pip install mcp[cli]

import asyncio
import mcp
from mcp.server import NotificationOptions, InitializationOptions

# ツール定義例
@mcp.server.tool("ツール名", "ツールの説明")
async def some_tool(param1: str, param2: int) -> str:
    # ツールのロジックを実装
    return "結果"

# サーバー初期化
server = mcp.server.Server()

# メイン関数
async def main():
    # stdin/stdoutストリームでサーバーを実行
    async with mcp.server.stdio.stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream, write_stream, 
            InitializationOptions(
                server_name="サーバー名",
                server_version="バージョン",
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={},
                ),
            ),
        )

if __name__ == "__main__":
    asyncio.run(main())
```

### TypeScript/JavaScript

```typescript
// 必要なライブラリをインストール
// npm install @modelcontextprotocol/sdk

import { Server, StdioServerTransport } from "@modelcontextprotocol/sdk";

// サーバーインスタンス作成
const server = new Server();

// ツール定義
const myTool = {
  name: "ツール名",
  description: "ツールの説明",
  parameters: {
    // パラメータ定義
  },
  execute: async (params) => {
    // ツールのロジック実装
    return { result: "結果" };
  }
};

// ツール登録
server.tools.registerTool(myTool);

// メイン関数
async function main() {
  // トランスポート設定
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("サーバー実行中");
}

main().catch((error) => {
  console.error("エラー:", error);
  process.exit(1);
});
```

## セキュリティ考慮事項

MCPサーバー実装時は以下のセキュリティ面を考慮する必要があります：

- **アクセス制御**: サーバーが公開するデータやツールへのアクセス制限
- **認証**: クライアントの認証メカニズム
- **データ保護**: 機密データの適切な取り扱い
- **リソース制限**: DoS攻撃防止のためのリソース使用制限

## ベストプラクティス

1. **明確なドキュメント**: 各ツール、リソース、プロンプトに詳細な説明を提供
2. **エラーハンドリング**: 適切なエラーメッセージとステータスコードを返す
3. **バージョニング**: 互換性のためのAPIバージョン管理
4. **テスト**: 単体テストと統合テストの実施
5. **ログ記録**: デバッグと監査のためのログ機能実装

## 既存のMCPサーバー例

- **ファイルシステム**: ファイル操作のためのセキュアなサーバー
- **PostgreSQL**: データベースアクセス用サーバー
- **GitHub**: リポジトリ管理やIssue管理機能を提供
- **Brave Search**: ウェブ検索機能を提供

## Claude Desktopとの接続

MCPサーバーをClaude Desktopと接続するには：

1. Claude Desktopをインストール
2. `~/Library/Application Support/Claude/claude_desktop_config.json`を編集
3. `mcpServers`セクションに自作サーバーを追加

```json
{
  "mcpServers": {
    "myserver": {
      "command": "実行コマンド",
      "args": ["引数1", "引数2"]
    }
  }
}
```

## デバッグとトラブルシューティング

1. **ログ出力**: 詳細なデバッグ情報を記録
2. **段階的テスト**: 基本機能から複雑な機能へ順次テスト
3. **エラーコード**: 明確なエラーコードとメッセージを実装
4. **MCP Inspector**: デバッグツールを利用した動作確認

## まとめ

MCPサーバーの実装により、さまざまなデータソースやツールをLLMに接続できます。これによりAIアシスタントの機能を拡張し、より豊かなユーザーエクスペリエンスを提供することが可能になります。
