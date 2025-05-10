" MCP Vim Plugin - Enhanced functionality for MCP files
" Maintainer: Claude
" Latest Revision: 2024-05-10

if exists("g:loaded_mcp_plugin")
  finish
endif
let g:loaded_mcp_plugin = 1

" Jump to the matching request/response with the same ID
function! MCPJumpToMatchingId()
  " Save the current position
  let save_cursor = getpos(".")
  let current_line = getline('.')
  
  " Extract the ID from the current line
  let matches = matchlist(current_line, '\"id\":\s*\(\d\+\)')
  if empty(matches)
    let matches = matchlist(current_line, '\"id\":\s*\"\([^\"]*\)\"')
  endif
  
  if !empty(matches)
    let id_value = matches[1]
    let search_pattern = '\"id\":\s*' . id_value
    if id_value =~ '^\d\+$'
      let search_pattern = '\"id\":\s*' . id_value . '\($\|[,}]\)'
    else
      let search_pattern = '\"id\":\s*\"' . id_value . '\"'
    endif
    
    " Determine if current line is a send or receive
    let is_send = current_line =~ '^mcp-send'
    let is_recv = current_line =~ '^mcp-recv'
    
    " Search for matching ID in the opposite direction
    if is_send
      " If we're on a send, look for the matching recv
      let match_line = search(search_pattern, 'nW')
      if match_line > 0
        " Make sure it's a recv line
        let match_text = getline(match_line)
        if match_text =~ '^mcp-recv'
          call cursor(match_line, 1)
          echo "Jumped to matching response (ID: " . id_value . ")"
          return
        endif
      endif
    elseif is_recv
      " If we're on a recv, look for the matching send
      let match_line = search(search_pattern, 'bnW')
      if match_line > 0
        " Make sure it's a send line
        let match_text = getline(match_line)
        if match_text =~ '^mcp-send'
          call cursor(match_line, 1)
          echo "Jumped to matching request (ID: " . id_value . ")"
          return
        endif
      endif
    endif
    
    " If we couldn't find a direct match, search in both directions
    let next_match = search(search_pattern, 'nW')
    let prev_match = search(search_pattern, 'bnW')
    
    if next_match > 0 && (prev_match == 0 || abs(next_match - line('.')) < abs(prev_match - line('.')))
      call cursor(next_match, 1)
      echo "Jumped to next occurrence of ID: " . id_value
    elseif prev_match > 0
      call cursor(prev_match, 1)
      echo "Jumped to previous occurrence of ID: " . id_value
    else
      echo "No matching ID found: " . id_value
      call setpos('.', save_cursor)
    endif
  else
    echo "No ID found on current line"
  endif
endfunction

" Jump to next ID
function! MCPJumpToNextId()
  let save_cursor = getpos(".")
  if search('\"id\":\s*\(\d\+\|\"[^\"]*\"\)', 'W')
    echo "Jumped to next ID"
  else
    echo "No more IDs found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to previous ID
function! MCPJumpToPrevId()
  let save_cursor = getpos(".")
  if search('\"id\":\s*\(\d\+\|\"[^\"]*\"\)', 'bW')
    echo "Jumped to previous ID"
  else
    echo "No previous IDs found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to next method
function! MCPJumpToNextMethod()
  let save_cursor = getpos(".")
  if search('\"method\":\s*\"[^\"]*\"', 'W')
    echo "Jumped to next method"
    normal! f"
  else
    echo "No more methods found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to previous method
function! MCPJumpToPrevMethod()
  let save_cursor = getpos(".")
  if search('\"method\":\s*\"[^\"]*\"', 'bW')
    echo "Jumped to previous method"
    normal! f"
  else
    echo "No previous methods found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to next request (mcp-send)
function! MCPJumpToNextRequest()
  let save_cursor = getpos(".")
  if search('^mcp-send', 'W')
    echo "Jumped to next request"
  else
    echo "No more requests found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to previous request (mcp-send)
function! MCPJumpToPrevRequest()
  let save_cursor = getpos(".")
  if search('^mcp-send', 'bW')
    echo "Jumped to previous request"
  else
    echo "No previous requests found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to next response (mcp-recv)
function! MCPJumpToNextResponse()
  let save_cursor = getpos(".")
  if search('^mcp-recv', 'W')
    echo "Jumped to next response"
  else
    echo "No more responses found"
    call setpos('.', save_cursor)
  endif
endfunction

" Jump to previous response (mcp-recv)
function! MCPJumpToPrevResponse()
  let save_cursor = getpos(".")
  if search('^mcp-recv', 'bW')
    echo "Jumped to previous response"
  else
    echo "No previous responses found"
    call setpos('.', save_cursor)
  endif
endfunction

" Search for specific ID value
function! MCPSearchId(...)
  if a:0 == 0
    let id = input('Enter ID to search for: ')
    if empty(id)
      echo "Search cancelled"
      return
    endif
  else
    let id = a:1
  endif
  
  let search_pattern = '\"id\":\s*' . id
  if id =~ '^\d\+$'
    let search_pattern = '\"id\":\s*' . id . '\($\|[,}]\)'
  else
    let search_pattern = '\"id\":\s*\"' . id . '\"'
  endif
  
  if search(search_pattern, 'w')
    echo "Found ID: " . id
  else
    echo "ID not found: " . id
  endif
endfunction

" Search for specific method
function! MCPSearchMethod(...)
  if a:0 == 0
    let method = input('Enter method to search for: ')
    if empty(method)
      echo "Search cancelled"
      return
    endif
  else
    let method = a:1
  endif
  
  let search_pattern = '\"method\":\s*\"' . method . '\"'
  if search(search_pattern, 'w')
    echo "Found method: " . method
  else
    echo "Method not found: " . method
  endif
endfunction

" Define commands
command! -nargs=? MCPFindId call MCPSearchId(<f-args>)
command! -nargs=? MCPFindMethod call MCPSearchMethod(<f-args>)
command! MCPJump call MCPJumpToMatchingId()
command! MCPNextId call MCPJumpToNextId()
command! MCPPrevId call MCPJumpToPrevId()
command! MCPNextMethod call MCPJumpToNextMethod()
command! MCPPrevMethod call MCPJumpToPrevMethod()
command! MCPNextRequest call MCPJumpToNextRequest()
command! MCPPrevRequest call MCPJumpToPrevRequest()
command! MCPNextResponse call MCPJumpToNextResponse()
command! MCPPrevResponse call MCPJumpToPrevResponse()

" Default key mappings (only activated for mcp filetype)
augroup MCPMappings
  autocmd!
  " Jump between matching IDs
  autocmd FileType mcp nnoremap <buffer> <leader>j :MCPJump<CR>

  " Jump to next/previous ID
  autocmd FileType mcp nnoremap <buffer> ]i :MCPNextId<CR>
  autocmd FileType mcp nnoremap <buffer> [i :MCPPrevId<CR>

  " Jump to next/previous method
  autocmd FileType mcp nnoremap <buffer> ]m :MCPNextMethod<CR>
  autocmd FileType mcp nnoremap <buffer> [m :MCPPrevMethod<CR>

  " Standard Vim function key mappings
  autocmd FileType mcp nnoremap <buffer> ]] :MCPNextMethod<CR>
  autocmd FileType mcp nnoremap <buffer> [[ :MCPPrevMethod<CR>

  " Jump to next/previous request
  autocmd FileType mcp nnoremap <buffer> ]r :MCPNextRequest<CR>
  autocmd FileType mcp nnoremap <buffer> [r :MCPPrevRequest<CR>

  " Jump to next/previous response
  autocmd FileType mcp nnoremap <buffer> ]p :MCPNextResponse<CR>
  autocmd FileType mcp nnoremap <buffer> [p :MCPPrevResponse<CR>

  " Search for ID/method
  autocmd FileType mcp nnoremap <buffer> <leader>i :MCPFindId<CR>
  autocmd FileType mcp nnoremap <buffer> <leader>m :MCPFindMethod<CR>
augroup END