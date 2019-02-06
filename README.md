# efm-langserver

General purpose Language Server that can use specified error message format
generated from specified command. This is useful for editing code with linter.

![efm](https://raw.githubusercontent.com/mattn/efm-langserver/master/screenshot.png)

## Usage

```text
Usage: efm-langserver [command...]
  -efm value
       errorformat
  -stdin
       use stdin
```

### Configuration for ERB with syntax check using erb command

```vim
augroup LspERB
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver-erb',
      \ 'cmd': {server_info->['efm-langserver', '-offset=1', '-stdin', &shell, &shellcmdflag, 'erb -x -T - | ruby -c']},
      \ 'whitelist': ['eruby'],
      \ })
augroup END
```

### Configuration for Vim script with syntax check using [vint](https://github.com/Kuniwak/vint)

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

### Configuration for Markdown with syntax check using [markdownlint-cli](https://github.com/igorshubovych/markdownlint-cli)

```vim
augroup LspMarkdown
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver-markdown',
      \ 'cmd': {server_info->['efm-langserver', '-efm=%f: %l: %m', '-stdin', &shell, &shellcmdflag, 'markdownlint -s']},
      \ 'whitelist': ['markdown'],
      \ })
augroup END
```

## Installation

```console
$ go get github.com/mattn/efm-langserver/cmd/efm-langserver
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
