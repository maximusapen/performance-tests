# FileOverload tool
FileOverload uses up all open file descriptors. It is intended to be used to test how a system behaves when files can't be opened.

## Running Tests
Build fileOverload and then run the script:
```
./fileOverload.sh [<delay>]
```

Where <delay> is the number of seconds the file descriptor are held.
