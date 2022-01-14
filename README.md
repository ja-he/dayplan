[![Go](https://github.com/ja-he/dayplan/actions/workflows/go.yml/badge.svg)](https://github.com/ja-he/dayplan/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ja-he/dayplan)](https://goreportcard.com/report/github.com/ja-he/dayplan)

# Dayplan

A utility to plan your day.

## Warning/Disclaimer

This is a project initially written to familiarize myself with go.
There will probably a lot of less-than-idiomatic (read: shitty) code in here for
quite a while to come (read: forever).
It's the middle ground between what I want to use and what I had time to make
and I don't see why anybody else would want to use this, at least for now.

## Installation

All that's really needed is a `go install github.com/ja-he/dayplan@latest`.  
_However_ there is a small build script `build.sh` available that takes care of
inserting version and commit information.
Therefore the recommended steps are:

    git clone https://github.com/ja-he/dayplan
    cd dayplan
    ./build.sh install

As it still uses `go install` under the hood, the binary should be in your
`$GOPATH` (or if empty in `$HOME/go/bin`).

## Usage

- Help messages are available e.g. via `-h`
- in TUI mode the key `?` toggles a help popup panel showing basic controls

### Regular TUI Usage

Dayplan mainly works as a terminal UI (TUI) program invoked simply by calling
the program without subcommand: `dayplan`.
In this mode it allows you sketch out the events of a day, similar to how a
graphical calendar application might work.
These events can then be shuffled around, resized, renamed, etc. as the day goes
on and it turns out that one task actually took a lot longer or that phone call
fell through. Thus you end up with a list of the (important) events of the day.
Make sure you use the `w` key to write the events of the day to its file.

### Getting Summaries

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

### Views

Using `ESC` you can switch from a single day's view to a week view, and from
there to the full month. You can still scroll through the days and events in
this mode but not perform and edits.

Using `i` you can step back into the day view.

### Configuration and Defaults

By default dayplan uses the directory `${HOME}/.config/dayplan` for
configuration and data storage. This directory can be set with the
`DAYPLAN_HOME` environment variable.
In the subdirectory `days` then days are stored as files named by
`YYYY-MM-DD` format.
Optionally, category styles can be defined in the file `category-styles.yaml`.

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

### Categories

Category styles  are formatted as YAML in in
`${DAYPLAN_HOME}/category-styles.yaml`.
Here's an example with two styles defined:
```yaml
- name: uni
  fg: '#000000'
  bg: '#ffdccc'
- name: work
  fg: '#000000'
  bg: '#ffcccc'
```
