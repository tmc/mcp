import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

const endpoint = process.argv[2];
if (!endpoint) {
  throw new Error("usage: node streamable_http_smoke.mjs http://127.0.0.1:port/mcp");
}

const transport = new StreamableHTTPClientTransport(new URL(endpoint));
const client = new Client({
  name: "tmc-mcp-typescript-sdk-streamable-http-smoke",
  version: "0.0.0",
});

try {
  await withTimeout(client.connect(transport), "initialize");

  const serverVersion = client.getServerVersion();
  if (serverVersion?.name !== "tmc-mcp-typescript-stdio-smoke") {
    throw new Error(`unexpected server name: ${serverVersion?.name}`);
  }
  if (!transport.sessionId) {
    throw new Error("missing streamable HTTP session ID");
  }

  const listed = await withTimeout(client.listTools({}), "tools/list");
  const toolNames = listed.tools.map((tool) => tool.name).sort();
  if (!toolNames.includes("echo")) {
    throw new Error(`echo tool missing from tools/list: ${toolNames.join(",")}`);
  }

  const called = await withTimeout(
    client.callTool({
      name: "echo",
      arguments: {
        message: "hello from streamable http",
      },
    }),
    "tools/call",
  );
  const text = called.content.find((item) => item.type === "text")?.text;
  if (text !== "echo: hello from streamable http") {
    throw new Error(`unexpected echo result: ${JSON.stringify(called)}`);
  }

  console.log(JSON.stringify({
    client: "@modelcontextprotocol/sdk",
    transport: "streamable-http",
    server: serverVersion.name,
    session: transport.sessionId,
    tools: toolNames,
    result: text,
  }));
} finally {
  if (transport.sessionId) {
    await withTimeout(transport.terminateSession(), "session/delete");
  }
  await client.close();
}

async function withTimeout(promise, label) {
  let timeout;
  try {
    return await Promise.race([
      promise,
      new Promise((_, reject) => {
        timeout = setTimeout(() => reject(new Error(`${label} timed out`)), 10_000);
      }),
    ]);
  } finally {
    clearTimeout(timeout);
  }
}
