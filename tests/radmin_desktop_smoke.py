"""Opt-in real adapter test of the production desktop API; never changes VPN/firewall.
Build tests/webui first. Default checks actual Radmin bind/invite/restart.
--messages additionally requires TCP communication between two local Radmin hosts;
this is not a test between two computers or houses.
"""
import argparse,base64,json,pathlib,socket,tempfile
from webui_smoke import Bridge

def free_port():
 with socket.socket() as s:
  s.bind(('127.0.0.1',0));return s.getsockname()[1]
def main():
 a=argparse.ArgumentParser();a.add_argument('binary');a.add_argument('--messages',action='store_true');args=a.parse_args()
 parent=pathlib.Path('data').resolve();parent.mkdir(exist_ok=True);processes=[]
 with tempfile.TemporaryDirectory(prefix='radmin-desktop-',dir=parent) as temp:
  root=pathlib.Path(temp)
  try:
   one=Bridge(pathlib.Path(args.binary).resolve(),root/'one');processes.append(one)
   state=one.rpc('State',ID='')
   adapters=[x for x in state['connectivity']['adapters'] if x['provider']=='radmin' and x['up'] and any(ip.startswith('26.') for ip in x['ipv4'])]
   if not adapters:print('SKIP: no active Radmin adapter with virtual IPv4');return
   if len(adapters)>1:raise AssertionError('Select one Radmin adapter explicitly for real-host QA')
   adapter=adapters[0];tcp=free_port()
   settings={**state['settings'],'mode':'radmin','interfaceId':adapter['id'],'port':tcp}
   one.rpc('SaveSettings',Settings=settings)
   state=one.rpc('State',ID='');sid=state['network']['id'];ip=state['connectivity']['ip']
   expected=f'/ip4/{ip}/tcp/{tcp}/p2p/{sid}'
   assert state['network']['localOnly'] and not state['networkError']
   assert state['connectivity']['selected']['id']==adapter['id'] and state['connectivity']['selected']['provider']=='radmin'
   assert state['network']['addresses']==[expected]
   raw=one.rpc('Invite');legacy=one.rpc('LegacyInvite',Text=raw);encoded=legacy.split(':',1)[1];invite=json.loads(base64.urlsafe_b64decode(encoded+'='*((-len(encoded))%4)))
   assert invite['scope']=='radmin' and invite['id']==sid and invite['addresses']==[expected] and not invite['relays']
   one.close();one=Bridge(pathlib.Path(args.binary).resolve(),root/'one');processes.append(one)
   state=one.rpc('State',ID='')
   assert state['settings']['mode']=='radmin' and state['settings']['interfaceId']==adapter['id'] and state['settings']['port']==tcp
   assert state['network']['id']==sid and state['network']['addresses']==[expected]
   print('PASS: real Radmin desktop selection, fixed listener, scoped invite, actual Peer ID, identity and settings after restart; other VPN not selected')
   if args.messages:
    two=Bridge(pathlib.Path(args.binary).resolve(),root/'two');processes.append(two)
    before=two.rpc('ConnectInvite',Text=one.rpc('Invite'))
    assert before['requiredMode']=='radmin' and not before['contactSaved']
    assert two.rpc('State',ID='')['settings'].get('mode','') in ('','auto')
    second={**settings,'port':free_port()};two.rpc('SaveSettings',Settings=second)
    result=two.rpc('ConnectInvite',Text=one.rpc('Invite'))
    assert result['connected'] and result['authenticated'] and result['targets'][0]['port']==tcp,result['message']
    two.rpc('Send',ID=sid,Text='Radmin desktop real adapter local QA')
    tid=two.rpc('State',ID='')['network']['id']
    assert one.rpc('State',ID=tid)['messages'][-1]['text']=='Radmin desktop real adapter local QA'
    one.close();one=Bridge(pathlib.Path(args.binary).resolve(),root/'one');processes.append(one)
    restarted=one.rpc('State',ID=tid);assert restarted['network']['id']==sid and restarted['messages'][-1]['text']=='Radmin desktop real adapter local QA'
    print('PASS: authenticated ping and durable text between two local Radmin desktop profiles (not two PCs)')
  finally:
   for process in processes:process.close()
if __name__=='__main__':main()
