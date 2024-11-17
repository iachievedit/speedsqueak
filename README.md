Dedicated to Squeak.  Rest in peace, sweet kitty.

# Installation

The OPS243 driver is written in Go.  Communications between
the different modules is via ZeroMQ.

```
sudo apt-get update && sudo apt-get install -y \
  golang \
  libzmq3-dev
```

## ZeroMQ Topics

| Publisher | Port | Event | Subscriber(s) |
|-----------|--|-----|---------------|
| `ops243` | 11205 | `event/speed`  | `camera` `uploader` | 
| `camera` | 11206 | `event/camera` | `uploader` |
| `heartbeat` | 11207 | `event/heartbeat` | `uploader` |

