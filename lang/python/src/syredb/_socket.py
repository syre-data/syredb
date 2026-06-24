from typing import Any
import json
import socket
import struct

LOCALHOST = "127.0.0.1"


def _recv_exact(sock, n) -> bytes:
    buf = b""
    while len(buf) < n:
        chunk = sock.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("socket closed")

        buf += chunk

    return buf


def _recv_message(sock) -> Any:
    header = _recv_exact(sock, 8)
    length = struct.unpack("<Q", header)[0]
    payload = _recv_exact(sock, length)
    return json.loads(payload)


def _query(content: dict[str, Any], sock: socket.socket) -> Any:
    data = json.dumps(content).encode()
    header = struct.pack("<Q", len(data))
    sock.sendall(header)
    sock.sendall(data)
    return _recv_message(sock)
