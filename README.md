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

## Usage

```
Usage:
  dayplan [OPTIONS]

Application Options:
  -d, --day=<file>    Specify the day to plan

Help Options:
  -h, --help          Show this help message
```

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
  fg: #000000
  bg: #ffdccc
- name: work
  fg: #000000
  bg: #ffcccc
```
