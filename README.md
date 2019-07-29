# Common

Contains a set of Go packages with minimal dependencies that are useful for integration projects.

## Build Instructions

The cleanest way to build in this package is to do the following:

1. Create an empty directory for your go workspace
   * ```➜  mkdir godemo &&  cd godemo```
1. Set your GOPATH to this directory
   * ```➜  export GOPATH=`pwd` ```
1. Get the repository using git or 'go get' (examples below)
   * Use git to obtain the repository
     * ```➜  git clone https://github.com/<username>/common-host-libs.git src/github.com/hpe-storage/common-host-libs```
   * Use 'go get' to obtain util
     * ```➜  go get -d github.com/hpe-storage/common-host-libs/util```
1. Change your working directory to the root of the repository
   * ```➜  cd src/github.com/hpe-storage/common-host-libs```
1. The tests are configured to run on linux/64, so set your GO OS
   * ```➜  export GOOS=darwin```
1. Build then entire repository to make sure everything compiles and tests
   * ```➜  make all```


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