test:
	mcp-replay -mock-client manual-npx-server-everything-stdio.4.mcp | mcpspy -v -- mcp-replay -mock-server manual-npx-server-everything-stdio.4.mcp 2>&1 |head -n20
