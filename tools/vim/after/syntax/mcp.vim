" MCP syntax file overrides
" This file is loaded after the main syntax file and makes sure highlighting works

" Force syntax highlighting to be applied
if exists("b:current_syntax")
  if b:current_syntax == "mcp"
    " Explicitly define the most important syntax groups again
    
    " Define the basic patterns
    syn match mcpSend "^mcp-send\s\+.*$" contains=mcpTimestamp,mcpJSON,mcpLineNumber
    syn match mcpRecv "^mcp-recv\s\+.*$" contains=mcpTimestamp,mcpJSON,mcpLineNumber
    syn match mcpTimestamp "\[\d\{4}-\d\{2}-\d\{2} \d\{2}:\d\{2}:\d\{2}\.\d\{3}\]" contained
    syn match mcpLineNumber "^\s*\d\+" 

    " JSON regions
    syn region mcpJSON start="{" end="}" contained contains=mcpString,mcpNumber,mcpBoolean,mcpNull,mcpMethod,mcpID,mcpResult,mcpParams,mcpError,mcpNotify,mcpKey

    " Method highlighting
    syn match mcpMethodKey /"method"\s*:/ contained
    syn match mcpMethod /"method"\s*:\s*"[^"]*"/ contained contains=mcpMethodKey,mcpMethodQuote,mcpMethodString
    
    " ID highlighting
    syn match mcpIDKey /"id"\s*:/ contained
    syn match mcpID /"id"\s*:\s*\d\+/ contained contains=mcpIDKey,mcpIDNumber
    syn match mcpID /"id"\s*:\s*"[^"]*"/ contained contains=mcpIDKey,mcpIDStringQuote,mcpIDString
    syn match mcpIDNumber /\d\+/ contained
    
    " Apply bold to method and ID
    hi mcpMethodString ctermfg=220 guifg=#ffd700 cterm=bold gui=bold
    hi mcpIDNumber ctermfg=226 guifg=#ffff00 cterm=bold gui=bold
    hi mcpIDString ctermfg=226 guifg=#ffff00 cterm=bold gui=bold
  endif
endif