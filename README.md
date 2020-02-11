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
    lint-command: 'vint --stdin-display-name ${INPUT} -'
    lint-stdin: true

  markdown:
    lint-command: 'markdownlint -s'
    lint-stdin: true
    lint-formats:
      - '%f:%l %m'

  yaml:
    lint-command: 'yamllint -f parsable -'
    lint-stdin: true
    lint-formats:
      - '%f:%l:%c: %m'
    env:
      - 'PYTHONIOENCODING=UTF-8'

  javascript:
    lint-command: 'eslint -f unix --stdin'
    lint-ignore-exit-code: true
    lint-stdin: true

  ruby:
    format-command: 'rufo'

  sh:
    lint-command: 'shellcheck -f tty -'
    lint-stdin: true

  go:
    lint-command: "golangci-lint run"

  php:
    lint-command: './vendor/bin/phpstan analyze --error-format raw --no-progress'
```

### Configuration for [vim-lsp](https://github.com/prabirshrestha/vim-lsp/)

```vim
augroup LspEFM
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver',
      \ 'cmd': {server_info->['efm-langserver', '-c=/path/to/your/config.yaml']},
      \ 'whitelist': ['vim', 'eruby', 'markdown', 'yaml'],
      \ })
augroup END
```

### Configuration for [coc.nvim](https://github.com/neoclide/coc.nvim)

coc-settings.json
```jsonc
  // languageserver
  "languageserver": {
    "efm": {
      "command": "efm-langserver",
      "args": [],
      // custom config path
      // "args": ["-c", "/path/to/your/config.yaml"],
      "filetypes": ["vim", "eruby", "markdown", "yaml"]
    }
  },
```

## Supported Lint tools

* [vint](https://github.com/Kuniwak/vint) for Vim script
* [markdownlint-cli](https://github.com/igorshubovych/markdownlint-cli) for Markdown

## Installation

```console
$ go get github.com/mattn/efm-langserver
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
