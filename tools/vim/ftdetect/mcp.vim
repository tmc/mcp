" MCP file type detection
au BufRead,BufNewFile *.mcp set filetype=mcp
au BufRead,BufNewFile *mcp-* set filetype=mcp

" Ensure syntax is loaded with filetype
augroup mcpfiletype
  autocmd!
  autocmd FileType mcp source $VIMRUNTIME/syntax/mcp.vim
  autocmd FileType mcp source $VIMRUNTIME/plugin/mcp.vim
  autocmd FileType mcp colorscheme mcpdark
augroup END