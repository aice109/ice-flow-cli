<div align="center">

# iflow

**A local-first productivity CLI: task planning, reminders, and a personal command dictionary.**

Tasks · Reminders · Fuzzy command search · Self-updating · macOS & Linux

</div>

---

`iflow` keeps your planning, your reminders, and the commands you can never
quite remember in one fast, offline binary. Everything lives in a single SQLite
file on your machine — no account, no sync, no telemetry.

```console
$ iflow find doctor
#1  3×
    flutter doctor
    Check the Flutter environment
    tags: flutter, env
    example: flutter doctor -v
```

## Features

- **Task planning** — priorities, due dates, projects, tags, notes, nested
  subtasks, a `tree` view and a `today` dashboard.
- **Reminders** — one-shot or recurring (`--every weekday --at 09:30`), with a
  background job so they fire even when the CLI is closed.
- **Command dictionary** — save a command with an explanation and tags, then
  recall it with fuzzy search (`iflow find doctor`) or run it (`iflow run doctor`).
- **Self-updating** — `iflow update` checks GitHub Releases, verifies the
  checksum, and replaces itself atomically.
- **Bilingual interface** — English and Simplified Chinese, switched with
  `--lang`, `IFLOW_LANG`, or the `lang` config key.
- **Local-first** — a single SQLite database, XDG-compliant paths, no network
  except for the update check.

## Install

### Install script (macOS / Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/aice109/ice-flow-cli/main/install.sh | sh
```

Installs to `$HOME/.local/bin` (override with `INSTALL_DIR`) and verifies the
release checksum.

### `go install`

```sh
go install github.com/aice109/ice-flow-cli/cmd/iflow@latest
```

### From source

```sh
git clone https://github.com/aice109/ice-flow-cli
cd ice-flow-cli
make build      # ./bin/iflow
make install
```

Then initialise and check that everything is wired up:

```sh
iflow init
iflow version
```

Enable shell completion:

```sh
# zsh
iflow completion zsh > "${fpath[1]}/_iflow"
# bash
iflow completion bash > /etc/bash_completion.d/iflow
```

## Quick start

### Plan your work

```sh
iflow task add "Write the design doc" --due "tomorrow 9am" --prio high --tag work
iflow task add "Ship v1.0" --project launch
iflow task add "Draft the spec" --project launch --parent 2   # nested subtask

iflow                # today: overdue, due today, in progress
iflow task list      # open tasks
iflow task tree      # hierarchy
iflow task start 3   # mark in progress
iflow task done 3    # complete
```

### Set reminders

```sh
iflow remind add "Standup" --every weekday --at 09:30
iflow remind add "Call mom" --at "tomorrow 19:00"
iflow remind add "Stretch" --in 45m

iflow service install   # fire them even when iflow is closed
iflow remind list
```

`iflow service install` registers a launchd agent on macOS or a systemd
`--user` timer on Linux (crontab fallback), which runs `iflow remind daemon
--tick` every minute.

### Build your command dictionary

```sh
iflow cmd add "flutter doctor" -d "Check the Flutter environment" -t flutter,env --example "flutter doctor -v"
iflow cmd add "docker system prune -af" -d "Remove unused docker data" -t docker

iflow find doctor    # fuzzy search across command, tags and description
iflow run doctor     # run the top match (with confirmation)
```

Search is intentionally forgiving — `iflow find flu`, `fltdr` and `doctor` all
find `flutter doctor`, and frequently used entries rise to the top.

### Keep iflow current

```sh
iflow update --check   # is there a newer release?
iflow update           # download, verify and install
```

A quiet check runs at most once a day (disable with
`iflow config set auto_update_check false`).

## Commands at a glance

| Command | Description |
| --- | --- |
| `iflow task add\|list\|edit\|rm\|tree\|today` | Manage tasks |
| `iflow task start\|done\|reopen <id>` | Change task status |
| `iflow remind add\|list\|rm\|snooze\|enable\|disable` | Manage reminders |
| `iflow remind daemon [--tick]` | Run the reminder scheduler |
| `iflow cmd add\|list\|edit\|rm\|import\|export` | Manage the command dictionary |
| `iflow find <query>` | Fuzzy-search the dictionary |
| `iflow run <query>` | Run the best match |
| `iflow service install\|uninstall\|status` | Background reminder job |
| `iflow update [--check]` | Self-update |
| `iflow config [path\|get\|set]` | Inspect or edit configuration |
| `iflow init` | Create config and database |

Full reference: [`docs/usage.md`](docs/usage.md).

## Configuration

Defaults live in `~/.config/iflow/config.toml`; data lives in
`~/.local/share/iflow/iflow.db`. Both honour `XDG_CONFIG_HOME` /
`XDG_DATA_HOME`, and `IFLOW_HOME` overrides everything (handy for testing).

```toml
date_format       = "2006-01-02"
default_priority  = 2
notify            = "auto"            # auto|osascript|notify-send|stdout|off
auto_update_check = true
github_repo       = "aice109/ice-flow-cli"
lang              = "auto"            # auto|en|zh-CN
```

Edit it directly, or with `iflow config set <key> <value>`.

### Language

The interface is available in English and Simplified Chinese:

```console
$ iflow --lang zh-CN remind list
ID  状态  下次                          重复                              标题
1   启用  2026-10-16 09:30 (10 小时后)  FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR  站会
```

Pick it with `--lang`, the `IFLOW_LANG` environment variable, or
`iflow config set lang zh-CN`. The default, `auto`, follows your locale
environment, so `LANG=zh_CN.UTF-8` is enough. Command names, flags and `--json`
keys stay in English for scriptability.

## Documentation

- [Usage reference](docs/usage.md) — every command and flag
- [Command dictionary guide](docs/command-dictionary.md) — building and importing your library
- [Architecture](docs/architecture.md) — how it is put together

## Development

```sh
make build       # build ./bin/iflow
make test        # unit tests
make lint        # gofmt check + go vet
make snapshot    # local goreleaser build (no publish)
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow and release
process.

## Project

- Changelog: [CHANGELOG.md](CHANGELOG.md)
- Code of Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- License: [MIT](LICENSE)
- Bundle identifier: `com.aice109.iceflow`

## License

[MIT](LICENSE) © 2026 aice109
