# Console User Interface utilities

**cui** is a Golang library that allows an application to run a terminal display _à-la-ncurses_.
No big fancy features, just selection of lists and display of items.

The application will query the items in a first field (TODO: with a ,
filters the fetched items with a kind of wildcard on the primary key of each item,
then presents the items passing the filter in a scrollable list,
to eventually dump the selected item in a side panel.

## Example

```shell
go run github.com/jfsmig/cui/examples/test-paths
```

```console
┌─Query───────────────────────────────────────────────────────────────────┐┌─Error─────────────────────────────────────┐
│/var/log                                                                 ││                                           │
└─────────────────────────────────────────────────────────────────────────┘│                                           │
┌─Filter──────────────────────────────────────────────────────────────────┐│                                           │
│.*                                                                       ││                                           │
└─────────────────────────────────────────────────────────────────────────┘└───────────────────────────────────────────┘
┌─Objects───────────┐┌─Detail──────────────────────────────────────────────────────────────────────────────────────────┐
│alternatives.log   ││{                                                                                                │
│alternatives.log.1 ││ "path": "alternatives.log.2.gz",                                                                │
│alternatives.log.2.││ "size": 611,                                                                                    │
│alternatives.log.3.││ "mode": 420,                                                                                    │
│alternatives.log.4.││ "ctime": "2022-10-26T00:45:43.90924422+02:00"                                                   │
│alternatives.log.5.││}                                                                                                │
│alternatives.log.6.││                                                                                                 │
│alternatives.log.7.││                                                                                                 │
│apport.log         ││                                                                                                 │
│apport.log.1       ││                                                                                                 │
│apport.log.2.gz    ││                                                                                                 │
│apport.log.3.gz    ││                                                                                                 │
│apport.log.4.gz    ││                                                                                                 │
│apport.log.5.gz    ││                                                                                                 │
│apport.log.6.gz    ││                                                                                                 │
└───────────────────┘└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## TODO

This is work in progress, however the subsequent actions have been identified:
- fix display quirks on small terminal windows
- make it responsive on all the fields
- document all that
