# efm-langserver

General purpose Language Server that can use specified error message format
generated from specified command. This is useful for editing code with linter.

![efm](https://raw.githubusercontent.com/mattn/efm-langserver/master/screenshot.png)

## Usage

```text
Usage of efm-langserver:
  -c string
        path to config.yaml
  -log string
        logfile
```

### Example for config.yaml

Location of config.yaml is:

* UNIX: `$HOME/.config/efm-langserver/config.yaml`
* Windows: `%APPDATA%\efm-langserver\config.yaml`

Below is example for config.yaml .

```yaml
languages:
  eruby:
    lint-command: 'erb -x -T - | ruby -c'
    lint-stdin: true
    lint-offset: 1
    format-command: 'htmlbeautifier'

  vim:
    lint-command: 'vint -'
    lint-stdin: true

  markdown:
    lint-command: 'markdownlint -s'
    lint-stdin: true
    lint-formats:
      - '%f: %l: %m'
```

### Configuration for [vim-lsp](https://github.com/prabirshrestha/vim-lsp/)

```vim
augroup LspEFM
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver-erb',
      \ 'cmd': {server_info->['efm-langserver', '-config=/path/to/your/config.yaml']},
      \ 'whitelist': ['eruby', 'markdown'],
      \ })
augroup END
```

## Supported Lint tools

* [vint](https://github.com/Kuniwak/vint) for Vim script
* [markdownlint-cli](https://github.com/igorshubovych/markdownlint-cli) for Markdown

## Installation

```console
$ go get github.com/mattn/efm-langserver/cmd/efm-langserver
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
