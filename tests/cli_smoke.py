"""Real-process smoke test. Usage: python tests/cli_smoke.py dist/projetop2p-windows-amd64.exe"""
import pathlib
import socket
import subprocess
import sys
import tempfile
import threading
import time


class Node:
    def __init__(self, binary, data, bootstraps=()):
        self.lines = []
        self.lock = threading.Lock()
        # Reserve a distinct TCP port for this launch; close immediately before use.
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        args = [str(binary), '--data', str(data), '--listen', f'/ip4/127.0.0.1/tcp/{port}']
        for addr in bootstraps:
            args.extend(['--bootstrap', addr])
        self.p = subprocess.Popen(args, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=subprocess.STDOUT, text=True, encoding='utf-8',
                                  creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        self.reader = threading.Thread(target=self.read, daemon=True)
        self.reader.start()
        try:
            self.id = self.wait('Peer ID: ').split('Peer ID: ', 1)[1].strip()
            self.addr = self.wait('Endereço: ').split('Endereço: ', 1)[1].strip()
        except BaseException:
            self.kill()
            raise

    def read(self):
        for line in self.p.stdout:
            with self.lock:
                self.lines.append(line.rstrip())

    def wait(self, text, timeout=45):
        end = time.monotonic() + timeout
        while time.monotonic() < end:
            with self.lock:
                for line in self.lines:
                    if text in line:
                        return line
            if self.p.poll() is not None:
                raise AssertionError(f'Process exited: {self.lines}')
            time.sleep(.05)
        raise AssertionError(f'Timeout waiting for {text!r}: {self.lines}')

    def command(self, text):
        self.p.stdin.write(text + '\n')
        self.p.stdin.flush()

    def kill(self):
        if self.p.poll() is None:
            self.p.kill()
        self.p.wait(timeout=10)
        self.reader.join(timeout=5)
        self.p.stdin.close()
        self.p.stdout.close()


def main():
    binary = pathlib.Path(sys.argv[1]).resolve()
    nodes = []
    temp_parent = pathlib.Path('data').resolve()
    temp_parent.mkdir(exist_ok=True)
    temp = tempfile.TemporaryDirectory(prefix='p2p-process-', dir=temp_parent)
    try:
        directory = temp.name
        if directory:
            root = pathlib.Path(directory)
            a = Node(binary, root / 'a'); nodes.append(a)
            # One unavailable bootstrap and a live fallback, with the same authenticated ID.
            dead = '/ip4/127.0.0.1/tcp/1/p2p/' + a.id
            b = Node(binary, root / 'b', [dead, a.addr]); nodes.append(b)
            c = Node(binary, root / 'c', [a.addr]); nodes.append(c)
            if '--legacy-dht' not in sys.argv[2:]:b.command('/connect '+c.addr)
            b.wait('Conectado: ' + c.id)
            b.command('/send ' + c.id + ' process-message-1')
            c.wait('process-message-1')
            b.wait('Recebimento confirmado')
            a.kill()
            time.sleep(.2)
            b.command('/send ' + c.id + ' bootstrap-offline')
            c.wait('bootstrap-offline')
            old_id, old_addr = c.id, c.addr
            c.kill()
            b.wait('Desconectado: ' + old_id)
            c = Node(binary, root / 'c', [b.addr]); nodes.append(c)
            assert c.id == old_id, 'Identity changed on restart'
            assert c.addr != old_addr, 'Changed-address scenario not exercised'
            c.wait('Conectado: ' + b.id)
            c.command('/send ' + b.id + ' recovered-new-address')
            b.wait('recovered-new-address')
            c.wait('Recebimento confirmado')
            print('PASS: three processes, '+('legacy DHT discovery' if '--legacy-dht' in sys.argv[2:] else 'explicit invite connection (provider DHT quarantined)')+', signed text, bootstrap fallback/outage, crash, identity reuse, changed address and recovery')
            for n in nodes:
                n.kill()
    finally:
        for n in nodes:
            n.kill()
        temp.cleanup()


if __name__ == '__main__':
    main()
