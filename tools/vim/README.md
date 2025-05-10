# MCP Vim Syntax Highlighting

This Vim plugin provides syntax highlighting and navigation tools for MCP (Model Context Protocol) spy/trace files.

## Features

- Highlights MCP send and receive messages
- Colorizes JSON-RPC elements like methods, IDs, and parameters
- Distinguishes errors and notifications
- Provides navigation functions for jumping between related requests/responses
- Includes a movement system for navigating through MCP files
- Custom color scheme optimized for MCP files

## Installation

### Manual Installation

Copy the contents of this directory to your Vim runtime path:

```bash
mkdir -p ~/.vim/
cp -r syntax ftdetect colors plugin ~/.vim/
```

### Using Plugin Manager (Vim-Plug)

Add to your `.vimrc`:

```vim
Plug 'tmc/mcp/tools/vim', {'rtp': 'tools/vim'}
```

## Usage

Once installed, any file with `.mcp` extension or files with `mcp-` prefix will automatically use the MCP syntax highlighting.

You can also manually set the file type:

```vim
:set filetype=mcp
```

### Quick Loading

To quickly load the plugin without installation:

```vim
:source /path/to/mcp/tools/vim/syntax/mcp.vim
:source /path/to/mcp/tools/vim/plugin/mcp.vim
:set filetype=mcp
```

### Navigation Commands

The plugin adds the following commands for navigating MCP files:

#### ID Navigation
- `:MCPJump` - Jump between matching request and response with the same ID
- `:MCPNextId` - Jump to the next ID
- `:MCPPrevId` - Jump to the previous ID
- `:MCPFindId [id]` - Search for a specific ID

#### Method Navigation
- `:MCPNextMethod` - Jump to the next method
- `:MCPPrevMethod` - Jump to the previous method
- `:MCPFindMethod [method]` - Search for a specific method

#### Message Navigation
- `:MCPNextRequest` - Jump to the next request (mcp-send)
- `:MCPPrevRequest` - Jump to the previous request
- `:MCPNextResponse` - Jump to the next response (mcp-recv)
- `:MCPPrevResponse` - Jump to the previous response

### Key Mappings

The following key mappings are available in MCP files:

#### ID Navigation
- `<leader>j` - Jump to matching request/response with the same ID
- `]i` - Jump to next ID
- `[i` - Jump to previous ID
- `<leader>i` - Search for an ID

#### Method Navigation 
- `]m` - Jump to next method
- `[m` - Jump to previous method
- `<leader>m` - Search for a method

#### Message Navigation
- `]r` - Jump to next request (mcp-send)
- `[r` - Jump to previous request
- `]p` - Jump to next response (mcp-recv)
- `[p` - Jump to previous response

### Optional Color Scheme

To use the included color scheme:

```vim
:colorscheme mcpdark
```

Or add to your `.vimrc`:

```vim
autocmd FileType mcp colorscheme mcpdark
```

## Example

This syntax highlighting is optimized for MCP spy/trace files that typically look like:

```
mcp-send [2024-05-10 14:32:15.123] {"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}
mcp-recv [2024-05-10 14:32:15.456] {"jsonrpc":"2.0","id":1,"result":{...}}
mcp-recv [2024-05-10 14:32:16.789] {"jsonrpc":"2.0","method":"notifications/logMessage","params":{...}}
```