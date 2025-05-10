" Vim syntax file
" Language: MCP (Model Context Protocol) spy/trace files
" Maintainer: Claude
" Latest Revision: 2024-05-10

if exists("b:current_syntax")
  finish
endif

" Define the basic patterns
syn match mcpSend "^mcp-send\s\+.*$" contains=mcpTimestamp,mcpJSON,mcpLineNumber
syn match mcpRecv "^mcp-recv\s\+.*$" contains=mcpTimestamp,mcpJSON,mcpLineNumber
syn match mcpTimestamp "\[\d\{4}-\d\{2}-\d\{2} \d\{2}:\d\{2}:\d\{2}\.\d\{3}\]" contained
syn match mcpLineNumber "^\s*\d\+" 

" JSON regions
syn region mcpJSON start="{" end="}" contained contains=mcpString,mcpNumber,mcpBoolean,mcpNull,mcpMethod,mcpID,mcpResult,mcpParams,mcpError,mcpNotify,mcpKey

" Basic JSON types
syn match mcpKey /"\w\+"\s*:/ contained contains=mcpKeyQuote
syn match mcpKeyQuote /"/ contained
syn region mcpString start=/"/ skip=/\\["\\]/ end=/"/ contained contains=mcpStringContent
syn match mcpStringContent /[^"\\]\+/ contained
syn match mcpNumber /-\?\d\+\(\.\d\+\)\?\([eE][+-]\?\d\+\)\?/ contained
syn keyword mcpBoolean true false contained
syn keyword mcpNull null contained

" Method highlighting
syn match mcpMethodKey /"method"\s*:/ contained
syn match mcpMethod /"method"\s*:\s*"[^"]*"/ contained contains=mcpMethodKey,mcpMethodQuote,mcpMethodString
syn match mcpMethodQuote /"/ contained
syn match mcpMethodString /[^"\\]\+/ contained

" ID highlighting
syn match mcpIDKey /"id"\s*:/ contained
syn match mcpID /"id"\s*:\s*\d\+/ contained contains=mcpIDKey,mcpIDNumber
syn match mcpID /"id"\s*:\s*"[^"]*"/ contained contains=mcpIDKey,mcpIDStringQuote,mcpIDString
syn match mcpIDNumber /\d\+/ contained
syn match mcpIDStringQuote /"/ contained
syn match mcpIDString /[^"\\]\+/ contained

" Params highlighting
syn match mcpParamsKey /"params"\s*:/ contained
syn region mcpParams start=/"params"\s*:/ end=/,\|}\|$/ contained contains=mcpParamsKey,mcpString,mcpNumber,mcpBoolean,mcpNull,mcpKey

" Result highlighting
syn match mcpResultKey /"result"\s*:/ contained
syn region mcpResult start=/"result"\s*:/ end=/,\|}\|$/ contained contains=mcpResultKey,mcpString,mcpNumber,mcpBoolean,mcpNull,mcpKey

" Content highlighting
syn match mcpContentKey /"contents"\s*:/ contained
syn region mcpContent start=/"contents"\s*:/ end=/,\|}\|$/ contained contains=mcpContentKey,mcpString,mcpNumber,mcpBoolean,mcpNull,mcpKey

" Error highlighting
syn match mcpErrorKey /"error"\s*:/ contained
syn region mcpError start=/"error"\s*:/ end=/,\|}\|$/ contained contains=mcpErrorKey,mcpString,mcpNumber,mcpBoolean,mcpNull,mcpKey

" Special notification methods
syn match mcpNotify /"method"\s*:\s*"notifications\/[^"]*"/ contained

" Highlight definitions
hi def link mcpSend Comment
hi def link mcpRecv Statement
hi def link mcpTimestamp Comment
hi def link mcpLineNumber LineNr

hi def link mcpKey Identifier
hi def link mcpKeyQuote Delimiter
hi def link mcpString String
hi def link mcpStringContent String
hi def link mcpNumber Number
hi def link mcpBoolean Boolean
hi def link mcpNull Constant

hi def link mcpMethodKey Type
hi def link mcpMethodQuote Delimiter
hi def link mcpMethodString Function

hi def link mcpIDKey Keyword
hi def link mcpIDNumber Special
hi def link mcpIDString Special
hi def link mcpIDStringQuote Delimiter

hi def link mcpParamsKey Structure
hi def link mcpResultKey Type
hi def link mcpContentKey Type
hi def link mcpErrorKey Error
hi def link mcpNotify Special

let b:current_syntax = "mcp"