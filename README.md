<div align="center">

# iceflow

**A local-first productivity CLI: task planning, reminders, and a personal command dictionary.**

Tasks · Reminders · Fuzzy command search · Self-updating · macOS & Linux

</div>

---

`iceflow` keeps your planning, your reminders, and the commands you can never
quite remember in one fast, offline binary. Everything lives in a single SQLite
file on your machine — no account, no sync, no telemetry.

```console
$ iceflow find doctor
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
  recall it with fuzzy search (`iceflow find doctor`) or run it (`iceflow run doctor`).
- **Self-updating** — `iceflow update` checks GitHub Releases, verifies the
  checksum, and replaces itself atomically.
- **Bilingual interface** — English and Simplified Chinese, switched with
  `--lang`, `ICEFLOW_LANG`, or the `lang` config key.
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
go install github.com/aice109/ice-flow-cli/cmd/iceflow@latest
```

### From source

```sh
git clone https://github.com/aice109/ice-flow-cli
cd ice-flow-cli
make build      # ./bin/iceflow
make install
```

Then initialise and check that everything is wired up:

```sh
iceflow init
iceflow version
```

Enable shell completion:

```sh
# zsh
iceflow completion zsh > "${fpath[1]}/_iceflow"
# bash
iceflow completion bash > /etc/bash_completion.d/iceflow
```

## Quick start

### Plan your work

```sh
iceflow task add "Write the design doc" --due "tomorrow 9am" --prio high --tag work
iceflow task add "Ship v1.0" --project launch
iceflow task add "Draft the spec" --project launch --parent 2   # nested subtask

iceflow                # today: tasks + reminders, overdue first
iceflow -n 7           # the same, for the next 7 days
iceflow task list      # open tasks
iceflow task tree      # hierarchy
iceflow task start 3   # mark in progress
iceflow task done 3    # complete
```

### Set reminders

```sh
iceflow remind add "Standup" --every weekday --at 09:30
iceflow remind add "Call mom" --at "tomorrow 19:00"
iceflow remind add "Stretch" --in 45m

iceflow service install   # fire them even when iceflow is closed
iceflow remind list
```

`iceflow service install` registers a launchd agent on macOS or a systemd
`--user` service on Linux (crontab fallback). The service keeps a resident
`iceflow remind daemon` running, which wakes exactly when the next reminder is
due instead of polling once a minute. Re-run `install` after moving or
upgrading the binary so the resident process uses the new executable.

### Build your command dictionary

```sh
iceflow cmd add "flutter doctor" -d "Check the Flutter environment" -t flutter,env --example "flutter doctor -v"
iceflow cmd add "docker system prune -af" -d "Remove unused docker data" -t docker

iceflow find doctor    # fuzzy search across command, tags and description
iceflow run doctor     # run the top match (with confirmation)
```

Search is intentionally forgiving — `iceflow find flu`, `fltdr` and `doctor` all
find `flutter doctor`, and frequently used entries rise to the top.

### Keep iceflow current

```sh
iceflow update --check   # is there a newer release?
iceflow update           # download, verify and install
```

A quiet check runs at most once a day (disable with
`iceflow config set auto_update_check false`).

### Back up your data

Everything lives in two local files: `config.toml` and `iceflow.db`. Back both up
into one archive, and restore them later:

```sh
iceflow backup                          # -> ./iceflow-backup-<timestamp>.tar.gz
iceflow backup ~/iceflow-2026-10-16.tar.gz
iceflow restore ~/iceflow-2026-10-16.tar.gz
```

The database is snapshotted with SQLite's `VACUUM INTO`, so backups are
consistent even while the background scheduler is running. Restore saves the
current state to a `iceflow-pre-restore-<timestamp>.tar.gz` archive first and
restarts the scheduler, so the daemon reopens the restored database.

## Commands at a glance

| Command | Description |
| --- | --- |
| `iceflow task add\|list\|edit\|rm\|tree\|today` | Manage tasks |
| `iceflow [-n days]` | Today's dashboard: tasks + reminders |
| `iceflow task start\|done\|reopen <id>` | Change task status |
| `iceflow remind add\|list\|rm\|snooze\|enable\|disable` | Manage reminders |
| `iceflow remind daemon [--tick]` | Run the reminder scheduler |
| `iceflow cmd add\|list\|edit\|rm\|import\|export` | Manage the command dictionary |
| `iceflow find <query>` | Fuzzy-search the dictionary |
| `iceflow run <query>` | Run the best match |
| `iceflow service install\|uninstall\|status` | Background reminder job |
| `iceflow backup [file]` / `iceflow restore <file>` | Back up and restore config + database |
| `iceflow update [--check]` | Self-update |
| `iceflow config [path\|get\|set]` | Inspect or edit configuration |
| `iceflow init` | Create config and database |

Full reference: [`docs/usage.md`](docs/usage.md).

## Configuration

Defaults live in `~/.config/iceflow/config.toml`; data lives in
`~/.local/share/iceflow/iceflow.db`. Both honour `XDG_CONFIG_HOME` /
`XDG_DATA_HOME`, and `ICEFLOW_HOME` overrides everything (handy for testing).

```toml
date_format       = "2006-01-02"
default_priority  = 2
notify            = "auto"            # auto|osascript|notify-send|stdout|off
auto_update_check = true
github_repo       = "aice109/ice-flow-cli"
lang              = "auto"            # auto|en|zh-CN
```

Edit it directly, or with `iceflow config set <key> <value>`.

### Language

The interface is available in English and Simplified Chinese:

```console
$ iceflow --lang zh-CN remind list
ID  状态  下次                          重复                              标题
1   启用  2026-10-16 09:30 (10 小时后)  FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR  站会
```

Pick it with `--lang`, the `ICEFLOW_LANG` environment variable, or
`iceflow config set lang zh-CN`. The default, `auto`, follows your locale
environment, so `LANG=zh_CN.UTF-8` is enough. Command names, flags and `--json`
keys stay in English for scriptability.

## Documentation

- [Usage reference](docs/usage.md) — every command and flag
- [Command dictionary guide](docs/command-dictionary.md) — building and importing your library
- [Architecture](docs/architecture.md) — how it is put together

## Development

```sh
make build       # build ./bin/iceflow
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
