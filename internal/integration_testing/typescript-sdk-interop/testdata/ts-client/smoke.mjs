import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

const server = process.argv[2];
if (!server) {
  throw new Error("usage: node smoke.mjs /path/to/server");
}

const transport = new StdioClientTransport({
  command: server,
  stderr: "pipe",
});
const stderr = [];
transport.stderr.on("data", (chunk) => {
  stderr.push(chunk.toString("utf8"));
});

const client = new Client({
  name: "tmc-mcp-typescript-sdk-smoke",
  version: "0.0.0",
});

try {
  await withTimeout(client.connect(transport), "initialize");

  const serverVersion = client.getServerVersion();
  if (serverVersion?.name !== "tmc-mcp-typescript-stdio-smoke") {
    throw new Error(`unexpected server name: ${serverVersion?.name}`);
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
        message: "hello from typescript",
      },
    }),
    "tools/call",
  );
  const text = called.content.find((item) => item.type === "text")?.text;
  if (text !== "echo: hello from typescript") {
    throw new Error(`unexpected echo result: ${JSON.stringify(called)}`);
  }

  console.log(JSON.stringify({
    client: "@modelcontextprotocol/sdk",
    transport: "stdio",
    server: serverVersion.name,
    tools: toolNames,
    result: text,
  }));
} catch (err) {
  const serverStderr = stderr.join("").trim();
  if (serverStderr) {
    console.error(serverStderr);
  }
  throw err;
} finally {
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
