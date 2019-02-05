# efm-langserver

General purpose Language Server that can use specified error message format generated from specified command. This is useful for editing code with linter.

## Usage

```
Usage: efm-langserver [command...]
  -efm value
    	errorformat
  -stdin
    	use stdin
```

### Configuration for ERB with syntax check using erb command.

```vim
augroup LspERB
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver-erb',
      \ 'cmd': {server_info->['efm-langserver', '-stdin', &shell, &shellcmdflag, 'erb -x -T - | ruby -c']},
      \ 'whitelist': ['eruby'],
      \ })
augroup END
```

### Configuration for Vim script with syntax check using vint.

```vim
augroup LspVim
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver-vim',
      \ 'cmd': {server_info->['efm-langserver', '-stdin', &shell, &shellcmdflag, 'vint -']},
      \ 'whitelist': ['vim'],
      \ })
augroup END
```

### Configuration for Perl with syntax check.

```vim
augroup LspPerl
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver-perl',
      \ 'cmd': {server_info->['efm-langserver', '-stdin', 'perl', '-c']},
      \ 'whitelist': ['perl'],
      \ })
augroup END
```

## Installation

```
$ go get github.com/mattn/efm-langserver/cmd/efm-langserver
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
