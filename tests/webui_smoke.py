"""Exercise the real desktop service via the QA-only loopback bridge, no mocks.
Build: go build -o data/webui.exe ./tests/webui
Run: python tests/webui_smoke.py data/webui.exe
"""
import json
import pathlib
import re
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request


class Bridge:
    def __init__(self, binary, profile):
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            port = s.getsockname()[1]
        self.base = f'http://127.0.0.1:{port}'
        self.p = subprocess.Popen([str(binary), '--data', str(profile), '--port', str(port)],
                                  stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                  creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        try:
            deadline = time.monotonic() + 30
            while True:
                try:
                    with urllib.request.urlopen(self.base, timeout=1) as r:
                        html = r.read().decode()
                    self.token = re.search(r"X-QA-Token':'([0-9a-f]+)'", html)[1]
                    assert 'src="/assets/' in html, 'Production frontend missing'
                    break
                except (OSError, TypeError):
                    if self.p.poll() is not None or time.monotonic() > deadline:
                        raise
                    time.sleep(.1)
        except BaseException:
            self.close()
            raise

    def rpc(self, method, **args):
        data = json.dumps({'Method': method, **args}).encode()
        req = urllib.request.Request(self.base + '/rpc', data=data, headers={
            'Content-Type': 'application/json', 'Origin': self.base, 'X-QA-Token': self.token})
        try:
            with urllib.request.urlopen(req, timeout=60) as r:
                return json.load(r)
        except urllib.error.HTTPError as e:
            raise AssertionError(json.load(e)) from e

    def close(self):
        if self.p.poll() is None:
            self.p.kill()
        self.p.wait(timeout=10)


def main():
    binary = pathlib.Path(sys.argv[1]).resolve()
    parent = pathlib.Path('data').resolve(); parent.mkdir(exist_ok=True)
    processes = []
    with tempfile.TemporaryDirectory(prefix='webui-', dir=parent) as temp:
        try:
            root = pathlib.Path(temp)
            a = Bridge(binary, root/'a'); processes.append(a)
            b = Bridge(binary, root/'b'); processes.append(b)
            settings = {'name': 'Alice QA', 'bootstrap': [], 'relays': [], 'assisted': True}
            a.rpc('SaveSettings', Settings=settings)
            ai = a.rpc('Invite'); bi = b.rpc('Invite')
            a.rpc('AddContact', Text=bi); b.rpc('AddContact', Text=ai)
            sa = a.rpc('State', ID=''); sb = b.rpc('State', ID='')
            deadline = time.monotonic() + 35
            while True:
                try:
                    a.rpc('Send', ID=sb['network']['id'], Text='QA backend real')
                    break
                except AssertionError:
                    if time.monotonic() > deadline:
                        raise
                    time.sleep(.2)
            state = b.rpc('State', ID=sa['network']['id'])
            assert state['messages'][-1]['text'] == 'QA backend real'
            assert state['contacts'][0]['name'] == 'Alice QA'
            before = state['network']['id']
            b.close()
            b = Bridge(binary, root/'b'); processes.append(b)
            state = b.rpc('State', ID=sa['network']['id'])
            assert state['network']['id'] == before
            assert state['contacts'] and state['messages'][-1]['text'] == 'QA backend real'
            # The bridge must not accept unauthenticated local requests.
            req = urllib.request.Request(b.base+'/rpc', data=b'{"Method":"State"}')
            try:
                urllib.request.urlopen(req, timeout=3)
                raise AssertionError('Unprotected QA bridge')
            except urllib.error.HTTPError as e:
                assert e.code == 403
            print('PASS: real desktop API, production assets, settings, signed invites, contacts, durable delivery, crash/restart and bridge access control')
        finally:
            for p in processes:
                p.close()


if __name__ == '__main__':
    main()
