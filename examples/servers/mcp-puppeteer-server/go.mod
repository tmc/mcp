module github.com/tmc/mcp/examples/servers/mcp-puppeteer-server

go 1.23.0

toolchain go1.24.3

require (
	github.com/chromedp/chromedp v0.10.0
	github.com/tmc/mcp v0.0.0
)

require (
	github.com/chromedp/cdproto v0.0.0-20240801214329-3f85d328b335 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/time v0.11.0 // indirect
)

replace github.com/tmc/mcp => ../../..
