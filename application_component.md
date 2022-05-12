- [Overview](#overview)
- [Components](#components)
  - [Boot](#boot)

<small><i><a href='http://ecotrust-canada.github.io/markdown-toc/'>Table of contents generated with markdown-toc</a></i></small>

# Overview

This document will go through various application components

# Components

## `internal/boot`

The boot package in `internal` is utilized to start the application. Everything in the boot process must complete successfully for the application to start. If it does not, the application will not start.

## `pkg/database`

The `database` package allows us to interact with a postgres DB. We utilize the interface to ensure we can interact with any `sql` database as well. I copied most of the code here from `vulcanize/go-ethereum`. Down the road, internal teams should be able to reference the package instead of copy pasting it and re-implementing it.

## `pkg/beaconclient`

This package will contain code to interact with the beacon client.

### Known Gaps

Known Gaps tracking is handled within this package. The columns are as follows:

- StartSlot - The start slot for known_gaps, inclusive.
- EndSlot - The end slot for known_gaps, inclusive.
- CheckedOut - Indicates if any process is currently processing this entry.
- ErrorMessage - Captures any error message that might have occurred when previously processing this entry.
- EntryTime - The time this range was added to the DB. This can help us catch ranges that have not been processed for a long time due to some error.
- EntryProcess - The entry process that added this process. Potential options are StartUp, Error, HeadGap.
  - This can help us understand how a specific entry was added. It can be useful for debugging the application.
  - StartUp - Gaps found when we started the application.
  - Error - Indicates that the entry was added due to an error with processing.
  - HeadGap - Indicates that gaps where found when keeping up with Head.

## `pkg/version`

A generic package which can be utilized to easily version our applications.

## `pkg/gracefulshutdown`

A generic package that can be used to shutdown various services within an application.

## `pkg/loghelper`

This package contains useful functions for logging.

## `internal/shutdown`

This package is used to shutdown the `ipld-ethcl-indexer`. It calls the `pkg/gracefulshutdown` package.
