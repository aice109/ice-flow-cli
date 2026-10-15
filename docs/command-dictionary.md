# Building your command dictionary

The command dictionary is the part of `iflow` that remembers the commands you
can never quite recall. You teach it a command once, with a short explanation,
and afterwards a partial query brings it back.

## The core loop

```sh
# Teach iflow a command once ...
iflow cmd add "flutter doctor" \
  -d "Check whether the Flutter toolchain is configured correctly" \
  -t flutter,env \
  --example "flutter doctor -v"

# ... and find it later with whatever you remember.
iflow find doctor
iflow find flutter
iflow find flu
```

All three queries return `flutter doctor` together with the explanation, tags
and example.

```console
$ iflow find flu
#1  3×
    flutter doctor
    Check whether the Flutter toolchain is configured correctly
    tags: flutter, env
    example: flutter doctor -v
```

The `3×` is a usage counter: entries you look up (or `iflow run`) rise in the
rankings over time.

## How search ranks results

1. An exact match on the command always wins.
2. Otherwise each candidate is scored against three fields, weighted
   `command` > `tags` > `description`.
3. Scores reward matches on word boundaries and consecutive characters, and
   penalize gaps — so `doctor` and `fltdr` both find `flutter doctor`, but the
   former scores higher.
4. The usage counter breaks ties.

The matched characters are highlighted in the terminal, so you can see *why* a
result matched.

## Running a saved command

```sh
iflow run doctor
```

`iflow run` resolves the query to the top match, asks for confirmation (skip
with `--yes`), executes it through `sh -c`, and bumps its usage count. Because
it uses your shell, pipes and redirections work:

```sh
iflow cmd add "git cleanup" -d "Prune merged branches" --example "git branch --merged | grep -v main | xargs git branch -d"
iflow run git cleanup
```

## Importing an existing library

### JSON

```json
[
  {
    "command": "flutter doctor",
    "description": "Check the Flutter environment",
    "tags": ["flutter", "env"],
    "example": "flutter doctor -v",
    "shell": "zsh"
  },
  {
    "command": "docker system prune -af",
    "description": "Remove all unused docker data",
    "tags": ["docker"]
  }
]
```

```sh
iflow cmd import commands.json
```

### Plain text

One command per line, with optional description and tags:

```
flutter doctor :: Check the Flutter environment :: flutter,env
docker system prune -af :: Remove unused docker data :: docker
kubectl get pods --all-namespaces :: List pods in every namespace :: k8s
```

### Markdown

Markdown list items with a backticked command are understood, so you can keep
your dictionary in a note and import it:

```markdown
## Flutter

- `flutter doctor` - Check the Flutter environment
- `flutter pub get` - Fetch dependencies
```

## Exporting and sharing

```sh
iflow cmd export --format md   > my-commands.md
iflow cmd export --format json > my-commands.json
```

Both formats round-trip through `iflow cmd import`, so exporting is a simple way
to back up or share a library.

## Editing and removing

```sh
iflow cmd list
iflow cmd edit 1 --desc "Updated explanation" --tag flutter,env,setup
iflow cmd rm 1
```
