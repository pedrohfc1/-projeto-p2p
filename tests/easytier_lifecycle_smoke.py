"""Optional Windows process cleanup test using actual QA API and pinned helpers."""
import pathlib,sys,tempfile,socket,ctypes,time
from webui_smoke import Bridge
binary=pathlib.Path(sys.argv[1]).resolve();helpers=str(pathlib.Path(sys.argv[2]).resolve());root=pathlib.Path('data').resolve();root.mkdir(exist_ok=True)
def port():
 with socket.socket() as s:s.bind(('127.0.0.1',0));return s.getsockname()[1]
k=ctypes.WinDLL('kernel32',use_last_error=True);k.OpenProcess.argtypes=[ctypes.c_ulong,ctypes.c_int,ctypes.c_ulong];k.OpenProcess.restype=ctypes.c_void_p;k.WaitForSingleObject.argtypes=[ctypes.c_void_p,ctypes.c_ulong];k.CloseHandle.argtypes=[ctypes.c_void_p]
def stopped(pid):
 h=k.OpenProcess(0x100000,False,pid)
 if not h:return True
 try:return k.WaitForSingleObject(h,0)==0
 finally:k.CloseHandle(h)
with tempfile.TemporaryDirectory(prefix='easytier-cleanup-',dir=root) as tmp:
 b=Bridge(binary,pathlib.Path(tmp)/'profile')
 try:
  identity=b.rpc('State',ID='')['network']['id'];cfg={'binaryDir':helpers,'network':'private-lifecycle-test-network','secret':'synthetic-test-secret-not-for-real-networks','virtualIp':'10.203.0.1','appPort':port(),'transportPort':port(),'socksPort':port(),'rpcPort':port(),'bindIp':'127.0.0.1','peers':[],'stun':[]}
  b.rpc('StartIntegratedLab',LabConfig=cfg);state=b.rpc('IntegratedLabState');assert state['running'] and state['id']==identity
  pid=state['pid'];b.rpc('StopIntegratedLab');assert stopped(pid),'normal shutdown orphaned helper'
  b.rpc('StartIntegratedLab',LabConfig=cfg);pid=b.rpc('IntegratedLabState')['pid'];b.close()
  deadline=time.monotonic()+5
  while not stopped(pid) and time.monotonic()<deadline:time.sleep(.1)
  assert stopped(pid),'host crash orphaned helper'
  print('PASS: persistent libp2p identity reused; normal stop and killed host leave no managed EasyTier helper')
 finally:b.close()
