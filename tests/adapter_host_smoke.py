"""Read real Windows adapter metadata and bind a host if Hamachi is active.
No VPN authentication/configuration, peer scan, firewall edit or public discovery.
"""
import json
import pathlib
import queue
import re
import subprocess
import sys
import tempfile
import threading
import time


def main():
    binary = pathlib.Path(sys.argv[1]).resolve()
    vpn = sys.argv[sys.argv.index('--vpn')+1] if '--vpn' in sys.argv else 'hamachi'
    prefix = {'hamachi':'25.', 'radmin':'26.'}[vpn]
    flags = getattr(subprocess, 'CREATE_NO_WINDOW', 0)
    check = subprocess.run([str(binary), '--interfaces'], capture_output=True, encoding='utf-8', timeout=20, creationflags=flags)
    assert check.returncode == 0, check.stderr
    items = json.loads(check.stdout)
    candidates = [(a, ip) for a in items if a['provider'] == vpn and a['up']
                  for ip in a['ipv4'] if ip.startswith(prefix)]
    if not candidates:
        print(f'SKIP: adapter enumeration succeeded; no active {vpn} IPv4 for host binding')
        return
    adapter, ip = candidates[0]
    parent = pathlib.Path('data').resolve(); parent.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='hamachi-bind-', dir=parent) as profile:
        p = subprocess.Popen([str(binary), '--data', profile, '--interface-ip', ip,
                              '--listen', f'/ip4/{ip}/tcp/0', '--daemon', '--status-every', '100ms'],
                             stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, encoding='utf-8', creationflags=flags)
        lines = queue.Queue()
        t = threading.Thread(target=lambda: [lines.put(s.rstrip()) for s in p.stdout], daemon=True); t.start()
        client = None; client_reader = None
        try:
            end = time.monotonic() + 20
            while time.monotonic() < end:
                line = lines.get(timeout=5)
                if line.startswith('{'):
                    state = json.loads(line)
                    assert state['localOnly'] and state['dhtPeers'] == 0
                    assert state['addresses'] and all(a.startswith(f'/ip4/{ip}/tcp/') for a in state['addresses'])
                    assert not state.get('relayService')
                    if '--messages' not in sys.argv[2:]:
                        print(f'PASS: real {vpn} metadata and interface-only TCP host binding; no public DHT/relay. Message connectivity was not tested.')
                        return
                    client = subprocess.Popen([str(binary), '--data', str(pathlib.Path(profile)/'client'),
                                               '--interface-ip', ip, '--listen', f'/ip4/{ip}/tcp/0'],
                                              stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                              text=True, encoding='utf-8', creationflags=flags)
                    client_lines = queue.Queue()
                    client_reader = threading.Thread(target=lambda: [client_lines.put(s.rstrip()) for s in client.stdout], daemon=True); client_reader.start()
                    client.stdin.write('/connect '+state['addresses'][0]+'\n')
                    client.stdin.write('/send '+state['id']+' virtual-interface-local-message\n'); client.stdin.flush()
                    until = time.monotonic()+15
                    while time.monotonic()<until:
                        received=lines.get(timeout=5)
                        if received.startswith('Mensagem de ') and 'virtual-interface-local-message' in received:
                            print(f'PASS: real {vpn} driver metadata, scoped TCP, authenticated local message delivery; no public DHT/relay; not a test between houses')
                            return
                    debug=[]
                    while not client_lines.empty():
                        value=client_lines.get()
                        debug.append(re.sub(r'(?:25|26)\.\d+\.\d+\.\d+','25.[redacted]',value))
                    raise AssertionError('No virtual-interface message; client diagnostics: '+str(debug))
                    return
            raise AssertionError('No host status')
        finally:
            if client is not None:
                client.kill();client.wait(timeout=10)
                if client_reader is not None:client_reader.join(timeout=5)
                client.stdin.close();client.stdout.close()
            p.kill(); p.wait(timeout=10); t.join(timeout=5); p.stdout.close()


if __name__ == '__main__':
    main()
