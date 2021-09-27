# efm-langserver

[![Actions Status](https://github.com/mattn/efm-langserver/workflows/CI/badge.svg)](https://github.com/mattn/efm-langserver/actions)

General purpose Language Server that can use specified error message format
generated from specified command. This is useful for editing code with linter.

![efm](https://raw.githubusercontent.com/mattn/efm-langserver/master/screenshot.png)

## Usage

```text
Usage of efm-langserver:
  -c string
        path to config.yaml
  -d    dump configuration
  -logfile string
        logfile
  -loglevel int
        loglevel (default 1)
  -q    Run quieter
  -v    Print the version
```

### Configuration

Configuration can be done with either a `config.yaml` file, or through
a [DidChangeConfiguration](https://microsoft.github.io/language-server-protocol/specification.html#workspace_didChangeConfiguration)
notification from the client.
`DidChangeConfiguration` can be called any time and will overwrite only provided
properties.

`DidChangeConfiguration` only supports V2 configuration and cannot set `LogFile`.


#### InitializeParams

Because the configuration can be updated on the fly, capabilities might change
throughout the lifetime of the server. To enable support for capabilities that will
be available later, set them in the [InitializeParams](https://microsoft.github.io/language-server-protocol/specification.html#initialize)

Example
```json
{
    "initializationOptions": {
        "documentFormatting": true,
        "hover": true,
        "documentSymbol": true,
        "codeAction": true,
        "completion": true
    }
}
```

### Example for config.yaml

Location of config.yaml is:

* UNIX: `$HOME/.config/efm-langserver/config.yaml`
* Windows: `%APPDATA%\efm-langserver\config.yaml`

Below is example for `config.yaml` for Windows.

```yaml
version: 2
root-markers:
  - .git/
lint-debounce: 1s
commands:
  - command: notepad
    arguments:
      - ${INPUT}
    title: メモ帳

tools:
  eruby-erb: &eruby-erb
    lint-debounce: 2s
    lint-command: 'erb -x -T - | ruby -c'
    lint-stdin: true
    lint-offset: 1
    format-stdin: true
    format-command: htmlbeautifier

  vim-vint: &vim-vint
    lint-command: 'vint -'
    lint-stdin: true
    lint-formats:
      - '%f:%l:%c: %m'

  make-checkmake: &make-checkmake
    lint-command: 'checkmake'
    lint-stdin: true

  markdown-markdownlint: &markdown-markdownlint
    lint-command: 'markdownlint -s -c %USERPROFILE%\.markdownlintrc'
    lint-stdin: true
    lint-formats:
      - '%f:%l %m'
      - '%f:%l:%c %m'
      - '%f: %l: %m'

  markdown-pandoc: &markdown-pandoc
    format-command: 'pandoc -f markdown -t gfm -sp --tab-stop=2'

  rst-pandoc: &rst-pandoc
    format-command: 'pandoc -f rst -t rst -s --columns=79'

  rst-lint: &rst-lint
    lint-command: 'rst-lint'
    lint-formats:
      - '%tNFO %f:%l %m'
      - '%tARNING %f:%l %m'
      - '%tRROR %f:%l %m'
      - '%tEVERE %f:%l %m'

  yaml-yamllint: &yaml-yamllint
    lint-command: 'yamllint -f parsable -'
    lint-stdin: true

  python-flake8: &python-flake8
    lint-command: 'flake8 --stdin-display-name ${INPUT} -'
    lint-stdin: true
    lint-formats:
      - '%f:%l:%c: %m'

  python-mypy: &python-mypy
    lint-command: 'mypy --show-column-numbers'
    lint-formats:
      - '%f:%l:%c: %trror: %m'
      - '%f:%l:%c: %tarning: %m'
      - '%f:%l:%c: %tote: %m'

  python-black: &python-black
    format-command: 'black --quiet -'
    format-stdin: true

  python-autopep8: &python-autopep8
    format-command: 'autopep8 -'
    format-stdin: true

  python-yapf: &python-yapf
    format-command: 'yapf --quiet'
    format-stdin: true

  python-isort: &python-isort
    format-command: 'isort --quiet -'
    format-stdin: true

  python-pylint: &python-pylint
    lint-command: 'pylint --output-format text --score no --msg-template {path}:{line}:{column}:{C}:{msg} ${INPUT}'
    lint-stdin: false
    lint-formats:
      - '%f:%l:%c:%t:%m'
    lint-offset-columns: 1
    lint-category-map:
      I: H
      R: I
      C: I
      W: W
      E: E
      F: E

  dockerfile-hadolint: &dockerfile-hadolint
    lint-command: 'hadolint'
    lint-formats:
      - '%f:%l %m'

  sh-shellcheck: &sh-shellcheck
    lint-command: 'shellcheck -f gcc -x'
    lint-source: 'shellcheck'
    lint-formats:
      - '%f:%l:%c: %trror: %m'
      - '%f:%l:%c: %tarning: %m'
      - '%f:%l:%c: %tote: %m'

  sh-shfmt: &sh-shfmt
    format-command: 'shfmt -ci -s -bn'
    format-stdin: true

  javascript-eslint: &javascript-eslint
    lint-command: 'eslint -f visualstudio --stdin --stdin-filename ${INPUT}'
    lint-ignore-exit-code: true
    lint-stdin: true
    lint-formats:
      - "%f(%l,%c): %tarning %m"
      - "%f(%l,%c): %rror %m"


  php-phpstan: &php-phpstan
    lint-command: './vendor/bin/phpstan analyze --error-format raw --no-progress'

  php-psalm: &php-psalm
    lint-command: './vendor/bin/psalm --output-format=emacs --no-progress'
    lint-formats:
      - '%f:%l:%c:%trror - %m'
      - '%f:%l:%c:%tarning - %m'

  html-prettier: &html-prettier
    format-command: './node_modules/.bin/prettier ${--tab-width:tabWidth} ${--single-quote:singleQuote} --parser html'

  css-prettier: &css-prettier
    format-command: './node_modules/.bin/prettier ${--tab-width:tabWidth} ${--single-quote:singleQuote} --parser css'

  json-prettier: &json-prettier
    format-command: './node_modules/.bin/prettier ${--tab-width:tabWidth} --parser json'

  json-jq: &json-jq
    lint-command: 'jq .'

  json-fixjson: &json-fixjson
    format-command: 'fixjson'

  csv-csvlint: &csv-csvlint
    lint-command: 'csvlint'

  lua-lua-format: &lua-lua-format
    format-command: 'lua-format -i'
    format-stdin: true

  blade-blade-formatter: &blade-blade-formatter
    format-command: 'blade-formatter --stdin'
    format-stdin: true

  mix_credo: &mix_credo
    lint-command: "mix credo suggest --format=flycheck --read-from-stdin ${INPUT}"
    lint-stdin: true
    lint-formats:
      - '%f:%l:%c: %t: %m'
      - '%f:%l: %t: %m'
    root-markers:
      - mix.lock
      - mix.exs

  any-excitetranslate: &any-excitetranslate
    hover-command: 'excitetranslate'
    hover-stdin: true

languages:
  eruby:
    - <<: *eruby-erb

  vim:
    - <<: *vim-vint

  make:
    - <<: *make-checkmake

  markdown:
    - <<: *markdown-markdownlint
    - <<: *markdown-pandoc

  rst:
    - <<: *rst-lint
    - <<: *rst-pandoc

  yaml:
    - <<: *yaml-yamllint

  python:
    - <<: *python-flake8
    - <<: *python-mypy
    # - <<: *python-autopep8
    # - <<: *python-yapf
    - <<: *python-black
    - <<: *python-isort

  dockerfile:
    - <<: *dockerfile-hadolint

  sh:
    - <<: *sh-shellcheck
    - <<: *sh-shfmt

  javascript:
    - <<: *javascript-eslint

  php:
    - <<: *php-phpstan
    - <<: *php-psalm

  html:
    - <<: *html-prettier

  css:
    - <<: *css-prettier

  json:
    - <<: *json-jq
    - <<: *json-fixjson
    # - <<: *json-prettier

  csv:
    - <<: *csv-csvlint

  lua:
    - <<: *lua-lua-format

  blade:
    - <<: *blade-blade-formatter

  elixir:
    - <<: *mix_credo

  =:
    - <<: *any-excitetranslate
```

If you want to debug output of commands:

```yaml
version: 2
log-file: /path/to/output.log
log-level: 1
```

### Example for DidChangeConfiguration notification

```json
{
    "settings": {
        "rootMarkers": [".git/"],
        "languages": {
            "lua": {
                "formatCommand": "lua-format -i",
                "formatStdin": true
            }
        }
    }
}
```

### Configuration for [vim-lsp](https://github.com/prabirshrestha/vim-lsp/)

```vim
augroup LspEFM
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'efm-langserver',
      \ 'cmd': {server_info->['efm-langserver', '-c=/path/to/your/config.yaml']},
      \ 'allowlist': ['vim', 'eruby', 'markdown', 'yaml'],
      \ })
augroup END
```

[vim-lsp-settings](https://github.com/mattn/vim-lsp-settings) provide installer for efm-langserver.

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

### Configuration for [elgot](https://github.com/joaotavora/eglot)

Add to eglot-server-programs with major mode you want.

```lisp
(with-eval-after-load 'eglot
  (add-to-list 'eglot-server-programs
    `(markdown-mode . ("efm-langserver"))))
```

### Configuration for [neovim builtin LSP](https://neovim.io/doc/user/lsp.html) with [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)

init.vim

```vim
lua << EOF
require "lspconfig".efm.setup {
    init_options = {documentFormatting = true},
    settings = {
        rootMarkers = {".git/"},
        languages = {
            lua = {
                {formatCommand = "lua-format -i", formatStdin = true}
            }
        }
    }
}
EOF
```

## Supported Lint tools

* [vint](https://github.com/Kuniwak/vint) for Vim script
* [markdownlint-cli](https://github.com/igorshubovych/markdownlint-cli) for Markdown

## Installation

```console
go install github.com/mattn/efm-langserver@latest
```

Homebrew
```console
brew install efm-langserver
```

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
