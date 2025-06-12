## WARNING

These are just my solutions for the challenges on codecrafters.io
None of this is production code, let alone bug free. It works, but at what cost?

### http-server-go

It's a working http server making use of TCP primitives instead of using the http package from std library. Great fun to build and learn more about the TCP protocol. We've GET and POST requests, support some headers, and gzip comperssion.

### shell-go
REPL shell with support for some builtins. The parser took a long time to get good at handling quotes and escapes, but once that was working correctly, I was left with a pretty good set for extending more builtins easily.

### bittorrent-go
This is an implementation of a bittorrent client in go
