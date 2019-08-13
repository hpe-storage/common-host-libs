# common-host-libs

Contains a set of Go packages with minimal dependencies that are useful for integration projects.

## Packages

### asupparser

Used to parse xml sent from tool kits in-band to DSD.

### cert

Provides SSL certificate functions

### chapi - Common Host API

CHAPI Client and Server

### connectivity

Generic http client

### docker/dockerlt - Docker Lightweight

A lightweight docker client

### docker/dockervol - Docker Volume Client

A client to talk to Docker Volume Plugins

### dockerplugin - Docker Volume Plugin Provider

A lightweight workflow manager that directs work to the container-provider and the CHAPI server

### jconfig

A simple json based config file utility

### jsonutil

Helper functions for dealing with json

### linux

Linux storage related functions

### model

A data model for CHAPI

### sgio

SCSI generic library

### stringformat

Helper functions for dealing with strings

### tunelinux

linux storage subsystem tuning engine

### util

Generic helpers

## Building the common-host-libs

Build instructions can be found at [BUILDING.md](BUILDING.md)
