# Layout - generate new project from layout

Heavily inspired by [cookiecutter](https://github.com/cookiecutter/cookiecutter) and [yeoman](https://yeoman.io), however layout offers additional features and bonuses:

- single binary without runtime dependencies, compiled for all major OS
- supports boolean variables ([yikes, cookicutter!](https://github.com/cookiecutter/cookiecutter/issues/126))
- supports conditional variables ([cookiecutter, I am again pointing to you](https://github.com/cookiecutter/cookiecutter/issues/1438))
- supports plain includes and conditional includes
- supports (and validates): string, boolean, list of strings, integer, float
- supports versioning in case you want to lock specific version of `layout`
- supports file source and remote Git repository
- supports multiple inline hooks (with portable shell) and templated hooks 
- hooks also supports condition :-)
- supports normal labeling for variables input (cookiecuter...)

I generally do not like competing with another open-source projects but this time
I would like to say that this project aiming to fix legacy cookiecutter's problems
and keep best of three worlds (including yeoman and Go).

The utility designed to be completely universal not just in terms of supported languages
and approaches, but also in terms of operational experience and can be used in restricted environment with the same
convenience as in public.

This project is stands on open-source atlantis shoulders:

- [MVDan's protable shell](https://mvdan.cc/sh/) which allows writing inline shell script regardless of OS
- [Tengo language](https://github.com/d5/tengo) which provides complete, simple and fast language for conditions
- [Go-git](https://github.com/go-git/go-git) which is basically embedded go-native Git client
- [Survey](https://github.com/AlecAivazis/survey) provides fancy terminal UI
- [Masterminds](https://github.com/Masterminds) for supporting tools

... and many many others. I love open-source, and this project is my payback.
That's why [license](LICENSE) for the project is Apache 2.0 which means that you may use code as you wish but please
state changes (for legal details please read LICENSE file).

## Architecture

```mermaid
sequenceDiagram
    User->>layout: new <repo> <destination> 
    layout->>repo: fetch recursively, depth 1
    repo->>layout: data
    layout->>User: display questions
    layout->>destination: copy and render content, execute hooks
```


## Tengo

Helpers:

- `has(seq, opt) -> bool` returns true if `seq` contains value `opt`. Mostly used for checking selected options (type: `list`)

