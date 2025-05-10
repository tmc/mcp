" MCP Color Scheme for Vim - Dark background
" Maintainer: Claude
" Latest Revision: 2024-05-10

set background=dark

if exists("syntax_on")
  syntax reset
endif

let g:colors_name = "mcpdark"

" MCP syntax highlighting
hi mcpSend        ctermfg=114 guifg=#87d787
hi mcpRecv        ctermfg=110 guifg=#87afd7
hi mcpTimestamp   ctermfg=240 guifg=#585858
hi mcpLineNumber  ctermfg=240 guifg=#585858

hi mcpKey         ctermfg=109 guifg=#87afaf
hi mcpKeyQuote    ctermfg=244 guifg=#808080
hi mcpString      ctermfg=180 guifg=#d7af87
hi mcpStringContent ctermfg=180 guifg=#d7af87
hi mcpNumber      ctermfg=81  guifg=#5fd7ff
hi mcpBoolean     ctermfg=173 guifg=#d7875f
hi mcpNull        ctermfg=168 guifg=#d75f87

hi mcpMethodKey   ctermfg=214 guifg=#ffaf00
hi mcpMethodQuote ctermfg=202 guifg=#ff5f00
hi mcpMethodString ctermfg=220 guifg=#ffd700 cterm=bold gui=bold

hi mcpIDKey       ctermfg=255 guifg=#eeeeee cterm=bold gui=bold
hi mcpIDNumber    ctermfg=226 guifg=#ffff00 cterm=bold gui=bold
hi mcpIDString    ctermfg=226 guifg=#ffff00 cterm=bold gui=bold
hi mcpIDStringQuote ctermfg=202 guifg=#ff5f00

hi mcpParamsKey   ctermfg=116 guifg=#87d7d7
hi mcpResultKey   ctermfg=120 guifg=#87ff87
hi mcpContentKey  ctermfg=120 guifg=#87ff87
hi mcpErrorKey    ctermfg=196 guifg=#ff0000
hi mcpNotify      ctermfg=171 guifg=#d75fff cterm=bold gui=bold