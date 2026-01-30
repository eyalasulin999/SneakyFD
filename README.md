# 🥷 SneakyFD 🥷

I had a vision to develop a stealth front-connect backdoor that doesn't listen at all.

Servers have many processes and services that already listen - Let's use them!

**SneakyFD** hijacks established connection sockets from other listening processes.

> **⚠ WIP & POC**

![](https://skillicons.dev/icons?i=golang,python,linux)

## General Flow

- Monitor `/proc/net/tcp` for new established connections
    - matched by destination port and source port range
- Hijack the target socket from the socket owner process
    - only duplicates the socket, don't send&recv anything
    - by syscall `pidfd_getfd` (Linux 5.6+)
- Check if the connection is marked (is this really our connection?)
    - connection marked by TCP options (currently MSS supported)
    - by syscall `getsockopt`

At this point we start sending beacons to the client with current state - we're sure this is our connection.

- Socket exclusivity
    - to avoid races on recv from the socket, we want to get rid of the socket owner process
    - By killing the socket owner process (for example `sshd` forks each new connection)
    - By waiting until the socket timed out by the socket owner process

## Backdoor Features

- shell (no pty allocation)
- file download/upload
- local/remote port forwarding

## Demo

![Demo2](assets/demo2.png)
![Demo1](assets/demo1.png)

## Configuration

configuraion file [config/config.go](config/config.go)

| Config | Type | Description | Default Value |
|-|-|-|-|
| LogLevel | `zerolog.Level` | logger level | `zerolog.InfoLevel` |
| DstPorts | `types.Ports` | list of destination ports to match | `types.Ports{types.FixedPort{Port: 22}}` |
| SrcPorts | `types.Ports` | list of source ports to match | `types.Ports{types.RangePort{MinPort: 1337, MaxPort: 2337}}` |
| CheckInterval | `time.Duration` | interval of monitoring new established connection sockets | `1 * time.Second` |
| Markers | `marker.Markers` | list of unique connection validators (aka markers) | `marker.Markers{marker.TCPOptionsMarker{MSS: 1337}}` |
| KillProcess | `bool` | kill the socket owner process or wait until the socket timed out by the socket owner process | `true` |
| WaitProcessTimeout | `time.Duration` | timeout for waiting until the socket timed out by the socket owner process | `3 * time.Minute` |
| BeaconMagic | `[]byte` | magic bytes for beacons messages | `[]byte{0xDE, 0xAD, 0xBE, 0xEF}` |
| BackdoorHashedPassword | `string` | hashed password for backdoor authentication | `"b7b3c1f7e43eaca2f6ce67038e8b91f01d1540bbead8d9db6333a3b4226a6abe" // SHA256 - password: "sneakyfd"` |
| BackdoorVersionBanner | `string` | backdoor version banner (transfered plaintext with default communication cover) | `"SneakyFD"` |
| BackdoorFallbackShell | `[]string` | shell command to use if not set by client side | `[]string{"/bin/sh -i"}` |
| BackdoorHostSignerPrivKey | `[]byte` | private key for host signer (AKA hostkey) | `[]byte(`-----BEGIN OPENSSH PRIVA...` |

#### Client Example

```python
client.config.host = "127.0.0.1"
client.config.dst_port = 22
client.config.src_ports = [(1337, 2337)]
client.config.markers = [TCPOptionsMarker(mss=1337)]

client.connect()

client.shell()

client.download_file(...)
client.upload_file(...)

client.local_forward(...)
client.remote_forward(...)
client.active_forwards
client.close_all_forwards()

client.disconnect()
```

## TODO

- support tcp6 (mapped addresses as well)
- make the code cleaner & error handling well
- stealth mode
- communication covers
- sessions managment
- fix multiple remote port forwards closing issue
- support Linux<5.6 (by ptrace?)
- SOCKS forwarding (just client side patching)
- ..
- .