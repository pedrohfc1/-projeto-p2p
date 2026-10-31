"""Real-process operator diagnostics test (no GUI). Run with the updated CLI path."""
import json
import pathlib
import queue
import subprocess
import sys
import tempfile
import threading
import time


def main():
    binary = pathlib.Path(sys.argv[1]).resolve()
    root = pathlib.Path('data').resolve(); root.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='relay-operator-', dir=root) as profile:
        p = subprocess.Popen([str(binary), '--data', profile, '--listen', '/ip4/127.0.0.1/tcp/0',
                              '--relay-server', '--public', '--daemon', '--status-every', '100ms'],
                             stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                             text=True, encoding='utf-8', creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        lines = queue.Queue()
        reader = threading.Thread(target=lambda: [lines.put(s.rstrip()) for s in p.stdout], daemon=True)
        reader.start()
        try:
            addr = None
            deadline = time.monotonic() + 20
            seen = []
            while time.monotonic() < deadline:
                line = lines.get(timeout=5); seen.append(line)
                if line.startswith('Endereço: /ip4/127.'):
                    addr = line.split('Endereço: ', 1)[1]
                if line.startswith('{') and json.loads(line).get('relayService', {}).get('enabled'):
                    break
            else:
                raise AssertionError(seen)
            assert addr, seen
            check = subprocess.run([str(binary), '--check-relay', addr], capture_output=True,
                                   encoding='utf-8', timeout=20,
                                   creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
            assert check.returncode == 0, check.stderr
            report = json.loads(check.stdout)
            assert report['reserved'] and report['voucherVerified'], report
            assert report['limitBytes'] == 1048576 and report['limitSeconds'] == 120
            deadline = time.monotonic() + 10
            while time.monotonic() < deadline:
                line = lines.get(timeout=5)
                if line.startswith('{') and json.loads(line).get('relayService', {}).get('reservationsAccepted', 0) > 0:
                    break
            else:
                raise AssertionError('Reservation not visible in operator counters')
            print('PASS: daemon survives EOF, live service status, signed reservation probe, quotas and operator counters')
        finally:
            p.kill(); p.wait(timeout=10); reader.join(timeout=5); p.stdout.close()


if __name__ == '__main__':
    main()
