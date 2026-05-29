# mping

A minimal multi-host ping monitor. Pings any number of hosts on a configurable interval and displays results on a single line with colour-coded status.

```
14:32:07 google.com 1.1.1.1 192.168.99.1
14:32:12 google.com 1.1.1.1 192.168.99.1
```
*(reachable hosts are green, unreachable are red)*

## Features

- Ping multiple hosts concurrently
- Green / red output for reachable / unreachable hosts
- ICMP without elevated privileges (no `sudo` required)
- Single static binary — no runtime dependencies
- macOS, Linux, Windows, FreeBSD

## Installation

**From source** (requires Go 1.22+):

```bash
go install github.com/fmattheus/mping@latest
```

**From a release binary** — download the appropriate binary from the [Releases](../../releases) page.

**Build locally:**

```bash
git clone https://github.com/fmattheus/mping
cd mping
go build -o mping .
```

## Usage

```
mping [flags] host [host ...]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--interval` | `-i` | `5` | Seconds between ping cycles. Accepts decimals (`1.5`) or duration strings (`500ms`, `2m`). |
| `--timeout` | `-t` | 90% of interval | Per-ping timeout. Same format as `--interval`. |
| `--theme` | `-T` | `default` | Output theme (see below). |

### Themes

| Theme | Colours | Symbols | Notes |
|-------|---------|---------|-------|
| `default` | Green / Red | No | |
| `symbols` | Green / Red | ✓ / ✗ | |
| `colorblind` | Blue / Yellow | ✓ / ✗ | Suitable for deuteranopia and protanopia |
| `mono` | None | ✓ / ✗ | No colour dependency |

### Examples

```bash
# Monitor three hosts with default 5s interval
mping google.com 1.1.1.1 192.168.1.1

# Faster polling
mping -i 1 google.com github.com

# Decimal interval
mping -i 2.5 google.com

# Custom timeout
mping -i 5 -t 2 google.com

# Colorblind-friendly theme
mping --theme colorblind google.com 1.1.1.1

# No colour (symbols only)
mping -T mono google.com 1.1.1.1
```

## Configuration

Default values can be set in a config file. CLI flags always override the config file.

**Location:**
| OS | Path |
|----|------|
| Linux | `~/.config/mping/config` |
| macOS | `~/Library/Application Support/mping/config` |
| Windows | `%AppData%\mping\config` |

**Format** — `key = value`, lines starting with `#` are comments:

```
# mping configuration
interval = 5
timeout = 4.5
theme = default
```

**Keys:**

| Key | Description |
|-----|-------------|
| `interval` | Seconds between ping cycles |
| `timeout` | Per-ping timeout in seconds (omit to use 90% of interval) |
| `theme` | Output theme: `default`, `symbols`, `colorblind`, `mono` |

The file is optional — missing keys fall back to built-in defaults.

## Platform notes

### Linux

mping uses unprivileged ICMP sockets (`SOCK_DGRAM IPPROTO_ICMP`), which requires the kernel to allow them for your user. Most modern distributions (Ubuntu 20.04+, Fedora, Debian 11+) permit this by default. If all hosts show red with a socket error, check:

```bash
cat /proc/sys/net/ipv4/ping_group_range
# Should show: 0	2147483647
```

To enable it for the current session:

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

To make it permanent, add `net.ipv4.ping_group_range = 0 2147483647` to `/etc/sysctl.conf`.

### macOS / BSD

Works for any user without extra configuration.

### Windows

Works for any user without elevation. Run from Windows Terminal or any modern terminal for colour support.

## Cross-compilation

Use the provided Makefile to build for all platforms at once:

```bash
make dist
```

Binaries are written to `dist/`. See `make help` for individual targets.

## License

MIT — see [LICENSE](LICENSE).
