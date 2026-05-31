import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { spawn } from "node:child_process";
import { PassThrough } from "node:stream";

class RawStdioTransport {
  constructor(command) {
    this.command = command;
    this.stderr = new PassThrough();
    this.stdout = "";
  }

  async start() {
    this.process = spawn(this.command, [], {
      stdio: ["pipe", "pipe", "pipe"],
    });
    this.process.stderr.pipe(this.stderr);
    this.process.on("error", (err) => {
      this.onerror?.(err);
    });
    this.process.on("close", () => {
      this.onclose?.();
    });
    this.process.stdout.on("data", (chunk) => {
      this.stdout += chunk.toString("utf8");
      this.processReadBuffer();
    });
  }

  async send(message) {
    if (!this.process?.stdin) {
      throw new Error("transport is not started");
    }
    await new Promise((resolve) => {
      if (this.process.stdin.write(JSON.stringify(message))) {
        resolve();
        return;
      }
      this.process.stdin.once("drain", resolve);
    });
  }

  async close() {
    const proc = this.process;
    this.process = undefined;
    if (!proc) {
      return;
    }
    const closed = new Promise((resolve) => proc.once("close", resolve));
    proc.stdin.end();
    await Promise.race([closed, delay(2000)]);
    if (proc.exitCode === null) {
      proc.kill("SIGTERM");
      await Promise.race([closed, delay(2000)]);
    }
    if (proc.exitCode === null) {
      proc.kill("SIGKILL");
    }
  }

  processReadBuffer() {
    while (true) {
      const end = findJSONObjectEnd(this.stdout);
      if (end < 0) {
        return;
      }
      const raw = this.stdout.slice(0, end);
      this.stdout = this.stdout.slice(end);
      try {
        this.onmessage?.(JSON.parse(raw));
      } catch (err) {
        this.onerror?.(err);
      }
    }
  }
}

const server = process.argv[2];
if (!server) {
  throw new Error("usage: node smoke.mjs /path/to/server");
}

const transport = new RawStdioTransport(server);
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

function findJSONObjectEnd(input) {
  let depth = 0;
  let inString = false;
  let escape = false;
  let started = false;
  for (let i = 0; i < input.length; i++) {
    const ch = input[i];
    if (!started) {
      if (/\s/.test(ch)) {
        continue;
      }
      if (ch !== "{") {
        throw new Error(`unexpected JSON-RPC response prefix: ${JSON.stringify(input.slice(0, i + 1))}`);
      }
      started = true;
      depth = 1;
      continue;
    }
    if (inString) {
      if (escape) {
        escape = false;
      } else if (ch === "\\") {
        escape = true;
      } else if (ch === "\"") {
        inString = false;
      }
      continue;
    }
    if (ch === "\"") {
      inString = true;
      continue;
    }
    if (ch === "{") {
      depth++;
      continue;
    }
    if (ch === "}") {
      depth--;
      if (depth === 0) {
        return i + 1;
      }
    }
  }
  return -1;
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
