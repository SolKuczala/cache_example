# Cache example
**Implemented with Go routines and Grpc.**

## Quick start
Once cloned, from root directory use target from Makefile

> cd cache_example

> make all

## Usage
client <command> [arguments]

Commands:

    - set             set the key along the value, i.e.: <set> [key] [value]
    - get             get the value from the provided key, i.e.: <get> [key]
    - cmpAndSet       cmpAndSet provided the key along the old value, set the new value, i.e.:
                      cmpAndSet <key> [oldvalue] [newValue]

Example:

Once `run-server` and `run-client` are done

> ./clienclient set A 1

to set a non existent key with a value

> ./clienclient get A

to get a value providing an existent key