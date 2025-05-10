" MCP filetype plugin
" Language: MCP (Model Context Protocol) spy/trace files
" Maintainer: Claude
" Latest Revision: 2024-05-10

if exists("b:did_ftplugin")
  finish
endif
let b:did_ftplugin = 1

" Ensure syntax highlighting is applied
if !exists("g:syntax_on")
  syntax enable
endif
syntax on

" Load syntax file if not loaded
if !exists("b:current_syntax") || b:current_syntax != "mcp"
  runtime! syntax/mcp.vim
endif

" Set custom color scheme
colorscheme mcpdark

" Define Vim standard function navigation
setlocal define=\\\"method\\\":\\s*\\\"
setlocal foldmethod=syntax
setlocal commentstring=#\ %s

" Define sections for paragraph movements
setlocal sections=^mcp-send,^mcp-recv

" Define what constitutes a sentence
setlocal iskeyword+=\":,-,_,/

" Make sure our plugin is loaded
if !exists("g:loaded_mcp_plugin") 
  runtime! plugin/mcp.vim
endif

" Buffer-local mappings
let b:undo_ftplugin = "setl def< cms< fdm< sect< isk<"