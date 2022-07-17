[![Go](https://github.com/ja-he/dayplan/actions/workflows/go.yml/badge.svg)](https://github.com/ja-he/dayplan/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ja-he/dayplan)](https://goreportcard.com/report/github.com/ja-he/dayplan)
![Total Lines](https://img.shields.io/tokei/lines/github/ja-he/dayplan?color=%23ffffff)

[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/ja-he/dayplan?color=%23f54d27&label=latest%20version%20%28by%20Git%20tag%29&logo=git&logoColor=white)](https://github.com/ja-he/dayplan/tags)

# Dayplan

A utility to plan and track your time.

## Warning/Disclaimer

- __format-wise, nothing is set in stone right now__  
  I would like to keep things simple, but I've already thought about going to a
  Markdown-ish format (like taskell uses) and I'll assume that any format
	changes that I can accommodate for myself with sed are fair game
- __it's unpolished__  
  there's definitely rough edges that I just ignore for now to tackle more
  relevant work
- __the code might suck?__  
  I wrote this to familiarize myself with Go; who knows if it worked?
	(the goreportcard rating up top is definitely way too lenient)

but also

- __is it any good?__  
  sure. I've been happily using it for months. :)

## Installation

### Arch Linux

[![AUR version](https://img.shields.io/aur/version/dayplan?color=1793d1&label=AUR%20version&logo=archlinux&logoColor=1793d1)](https://aur.archlinux.org/packages/dayplan/)
[![AUR last modified](https://img.shields.io/aur/last-modified/dayplan?color=1793d1&label=AUR%20recency)](https://aur.archlinux.org/cgit/aur.git/log/?h=dayplan)

For Arch Linux and its derivatives there is an
[AUR package](https://aur.archlinux.org/packages/dayplan/) available.

### Manually

All that's really needed is a `go install github.com/ja-he/dayplan@latest`.  
_However_ there is a small build script `.scripts/build.sh` available that takes
care of inserting version and commit information.
Therefore the recommended steps are:

    git clone https://github.com/ja-he/dayplan
    cd dayplan
    ./.scripts/build.sh install

As it still uses `go install` under the hood, the binary should be in your
`$GOPATH` (or if empty in `$HOME/go/bin`).

## Usage

- Help messages are available e.g. via `-h`
- in TUI mode the key <kbd>?</kbd> toggles a help popup panel showing the
  controls based on the current context (this is generated; while being maybe
  not structured as nicely, it should generally provide the most complete and
  up-to-date information

### Regular TUI Usage (`tui`)

Dayplan mainly works as a terminal UI (TUI) program invoked simply by calling
the program with the `tui` subcommand:

    $ dayplan tui

In this mode it allows you sketch out the events of a day, similar to how a
graphical calendar application might work.

These events can then be shuffled around, resized, renamed, etc. as the day goes
on and it turns out that one task actually took a lot longer or that phone call
fell through. Thus you end up with a list of the (important) events of the day.

Dayplan can be controlled via both mouse and keyboard.
Key mappings are "vim-ish" and not currently configurable.

#### Keyboard-driven

| __key input__                                                      | ___does...___                                                              |
| :-:                                                                | :--                                                                        |
| <kbd>?</kbd>                                                       | open help (context-based)                                                  |
|                                                                    |                                                                            |
| <kbd>h</kbd> / <kbd>l</kbd>                                        | switch the current day                                                     |
| <kbd>i</kbd> / <kbd>ESC</kbd>                                      | switch between day, week, and month view                                   |
| <kbd>+</kbd> / <kbd>-</kbd>                                        | zoom in or out                                                             |
| <kbd>j</kbd> / <kbd>k</kbd>                                        | select next or previous event                                              |
| <kbd>d</kbd>                                                       | delete the current event                                                   |
|                                                                    |                                                                            |
| <kbd>CTRL-w</kbd><kbd>h</kbd> / <kbd>CTRL-w</kbd><kbd>l</kbd>      | switch to left / right ui pane                                             |
| <kbd>S</kbd>                                                       | toggle a summary view (for day/week/...)                                   |
| <kbd>W</kbd>                                                       | load the weather (see [the config section](#configuration-and-defaults))   |
|                                                                    |                                                                            |
| <kbd>w</kbd>                                                       | write the current day to file                                              |
| <kbd>q</kbd>                                                       | quit                                                                       |
|                                                                    |                                                                            |
| <kbd>m</kbd>                                                       | enter event move mode, in which...                                         |
| <kbd>j</kbd> / <kbd>k</kbd>                                        | ...move event up or down                                                   |
| <kbd>m</kbd> / <kbd>ESC</kbd>                                      | ...exit mode                                                               |
|                                                                    |                                                                            |
| <kbd>r</kbd>                                                       | enter event resize mode, in which...                                       |
| <kbd>j</kbd> / <kbd>k</kbd>                                        | ...lengthen or shorten event                                               |
| <kbd>r</kbd> / <kbd>ESC</kbd>                                      | ...exit mode                                                               |
|                                                                    |                                                                            |
| _(see help..._                                                     | _...for more)_                                                             |

#### Mouse-driven

To roughly emulate the expected behavior of a familiar calendar application, the
mouse can also be used for editing (in truth, this was the initial method
implemented, as it's pretty straightforward to define what the basic operations
should be and how they should behave):

- __move__: left click inside of an event and drag it
- __resize__: left click on the end (timestamp) of an event and drag it
- __edit name__: left click on the events name (or anywhere at the top of the event)
- __delete__: middle click on the event
- __split__: right click on the event at the time at which to split it

### Getting Summaries (`summarize`)

To then get summary information about the information generated in this way,
dayplan has the subcommand `summarize`. It requires you to specify a `--from`
and a `--til` date and summarizes the duration of events in this range by their
categories.
You could for example use dayplan as above to track your working hours and then
see how much you've worked in November 2021 using the following command:

```sh
$ dayplan summarize --from 2021-11-01 --til 2021-11-30 \
                    --category-filter work \
                    --human-readable
```
### Adding events via CLI (`add`)

Besides being able to add events in the TUI mode, events can also be added via
the `add` subcommand.
This is especially useful for adding repeat events, in which case a start date,
an end date, and the period of repetition need to be specified.

For more see `dayplan add -h`.

### Configuration and Defaults

By default dayplan uses the directory `${HOME}/.config/dayplan` for
configuration and data storage. This directory can be set with the
`DAYPLAN_HOME` environment variable.
In the subdirectory `days` then days are stored as files named by
`YYYY-MM-DD` format.
Optionally, category styles can be defined in the file `config.yaml`; also see
the [Configuration section](#configuration).

To get weather and sunrise/-set information you'll need to define latitude and
longitude as environment variables (e.g. in the `.bashrc`):
```
export LATITUDE=12.3456
export LONGITUDE=11.1111
```
For getting the weather information from OWM you'll also need to have
`OWM_API_KEY` defined in the same way, e.g.
```
export OWM_API_KEY=<key>
```

## File Content Formatting

### Days

A day, usually at `${DAYPLAN_HOME}/days/<YYYY-MM-DD>`, is a list of events
formatted as
```
<start>|<end>|<category>|<title>
```
so for example a day with three events might be
```
08:00|08:30|eat|Breakfast
08:30|10:00|fitness|Work Out
10:00|18:00|misc|Do Absolutely Nothing
```

## Configuration

Dayplan can be optionally configured in `${DAYPLAN_HOME}/config.yaml`.
Configuration currently entails theming and categories.

- The general UI colors are defined under `stylesheet`.
- The categories are listed under `categories`

Here a very short[^longer-example] example of the file format:
```yaml
stylesheet:
  normal:            { fg: '#000000', bg: '#ffffff' }
  timeline-day:      { fg: '#c0c0c0', bg: '#ffffff' }
  timeline-night:    { fg: '#f0f0f0', bg: '#000000' }
  timeline-now:      { fg: '#ffffff', bg: '#ff0000' }
  summary-default:   { fg: '#000000', bg: '#ffffff' }
  summary-title-box: { fg: '#000000', bg: '#f0f0f0', style: { italic: true } }

categories:
  - name: uni
    color: '#ffdccc'
  - name: work
    color: '#ffcccc'
    goal: # this helps track overtime
      workweek:
        monday:    5h
        tuesday:   5h
        wednesday: 5h
        thursday:  5h
        friday:    0h
        saturday:  0h
        sunday:    0h

  - name: cooking
    color: '#ccffe6'
  # ...
```

[^longer-example]:
    As these identifiers are very much subject to change pre-v1.0.0, I'm
    refraining from providing a default 'config.yaml' file in the repo.
    Please just look through the sources in 'src/config/config.go' to see the
    yaml-identifiers that are available for stylings.

> #### Note: Terminal Color Support
>
> Currently, a true color terminal is required; use of other color codes is not
> supported.  
> However, this is not inherent to dayplan; see #24.
