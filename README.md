## Go ping-er library

This is _yet another_ ping library for golang. I wrote this while playing with
the language, so as the [license](LICENSE) says there is no guarantee it will
actually work.

A simple [`ping`](cmd/ping) program is provided as an example, and can be
compiled by calling `make`.

### Limitations

Sending/receiving ICMP packets requires privileged access (so we can get
raw sockets). A program using this library needs:

- to be run as the `root` user OR
- to have CAP_NET_RAW set (`sudo setcap cap_net_raw+ep <binary>`) OR
- to have the net.ipv4.ping_group_range sysctl modified to include the group id
used when running in the specified range (for more information, see `man 7 icmp`).

The library automatically detects and uses what's available, and will return
a `socket: permission denied` error when calling `pinger.New()` if none
of the conditions above are met.
