"""Three real production desktop services. Optional actual Radmin adapter, no router/firewall changes.
This tests backend/API and production assets, not native window interaction or two houses.
"""
import argparse,json,pathlib,socket,tempfile,time
from webui_smoke import Bridge

def port():
 with socket.socket() as s:s.bind(('127.0.0.1',0));return s.getsockname()[1]
def main():
 ap=argparse.ArgumentParser();ap.add_argument('binary');ap.add_argument('--radmin',action='store_true');ap.add_argument('--metrics');args=ap.parse_args()
 binary=pathlib.Path(args.binary).resolve();parent=pathlib.Path('data').resolve();parent.mkdir(exist_ok=True);processes=[]
 def configure(b,name):
  state=b.rpc('State',ID='');items=state['connectivity']['adapters']
  if args.radmin:items=[i for i in items if i['provider']=='radmin' and i['up'] and any(ip.startswith('26.') for ip in i['ipv4'])];mode='radmin'
  else:items=[i for i in items if i['up'] and '127.0.0.1' in i['ipv4']];mode='interface'
  if not items:return False
  assert len(items)==1,'Choose a single test adapter explicitly'
  b.rpc('SaveSettings',Settings={'name':name,'mode':mode,'interfaceId':items[0]['id'],'port':port(),'assisted':False,'bootstrap':[],'relays':[]});return True
 def reject(b,method,**params):
  try:b.rpc(method,**params)
  except AssertionError:return
  raise AssertionError('Unauthorized operation accepted: '+method)
 started=time.monotonic()
 with tempfile.TemporaryDirectory(prefix='communities-',dir=parent) as folder:
  root=pathlib.Path(folder)
  try:
   owner=Bridge(binary,root/'owner');processes.append(owner)
   if not configure(owner,'Dona QA'):print('SKIP: requested adapter not available');return
   mod=Bridge(binary,root/'mod');processes.append(mod);assert configure(mod,'Moderador QA')
   member=Bridge(binary,root/'member');processes.append(member);assert configure(member,'Membro QA')
   owner.rpc('State',ID='');time.sleep(1)
   idle=owner.rpc('State',ID='')['metrics']
   group=owner.rpc('CreateCommunity',Name='Bairro conectado QA',Description='Três nós reais, sem servidor global',Icon='🇧🇷')
   invitation=owner.rpc('CommunityInvite',ID=group);joined=mod.rpc('JoinCommunity',Text=invitation);assert joined['status']=='participante',joined
   mid=mod.rpc('State',ID='')['network']['id']
   owner.rpc('CommunityAction',ID=group,Kind='member.role',Body={'target':mid,'role':'moderator'})
   mod.rpc('SyncCommunity',ID=group)
   invitation=mod.rpc('CommunityInvite',ID=group);joined=member.rpc('JoinCommunity',Text=invitation);assert joined['status']=='participante',joined
   mod.rpc('SyncCommunity',ID=group)
   mod.rpc('CommunityAction',ID=group,Kind='channel.create',Body={'name':'novidades','channelKind':'text'})
   member.rpc('SyncCommunity',ID=group);state=member.rpc('CommunityState',ID=group,Channel='geral',Before=0)
   channel=next(ch['id'] for ch in state['community']['channels'] if ch['name']=='novidades')
   assert any(ch['kind']=='voice' for ch in state['community']['channels'])
   reject(member,'CommunityAction',ID=group,Kind='member.role',Body={'target':member.rpc('State',ID='')['network']['id'],'role':'owner'})
   begin=time.perf_counter();sent=member.rpc('CommunityAction',ID=group,Kind='message.text',Body={'channelId':channel,'text':'<script>não executar</script> 🇧🇷'});roundtrip=(time.perf_counter()-begin)*1000
   assert sent['status']=='confirmada',sent
   mod.rpc('SyncCommunity',ID=group);received=mod.rpc('CommunityState',ID=group,Channel=channel,Before=0)
   assert len(received['messages'])==1 and received['messages'][0]['id']==sent['id']
   # Outage and crash/restart preserve ownership and immutable pending operation IDs.
   oid=owner.rpc('State',ID='')['network']['id'];owner.close()
   pending=member.rpc('CommunityAction',ID=group,Kind='message.text',Body={'channelId':channel,'text':'pendente durante queda'});assert pending['status']=='pendente',pending
   owner=Bridge(binary,root/'owner');processes.append(owner);assert owner.rpc('State',ID='')['network']['id']==oid
   deadline=time.monotonic()+35
   while True:
    try:
     member.rpc('SyncCommunity',ID=group);rows=member.rpc('CommunityState',ID=group,Channel=channel,Before=0)['messages']
     if any(m['id']==pending['id'] and m['status']=='confirmada' for m in rows):break
    except AssertionError:pass
    if time.monotonic()>deadline:raise AssertionError('Community outbox recovery timed out')
    time.sleep(.3)
   member.rpc('SyncCommunity',ID=group);rows=owner.rpc('CommunityState',ID=group,Channel=channel,Before=0)['messages'];assert len(rows)==2 and len({m['id'] for m in rows})==2
   # Direct messages continue using the existing text protocol/profile.
   owner.rpc('AddContact',Text=member.rpc('Invite'));member.rpc('AddContact',Text=owner.rpc('Invite'))
   owner.rpc('Send',ID=member.rpc('State',ID='')['network']['id'],Text='chat individual preservado')
   assert member.rpc('State',ID=oid)['messages'][-1]['text']=='chat individual preservado'
   diagnostics=member.rpc('TestConnectivity',ID=oid);assert diagnostics['connected']
   metrics={label:b.rpc('State',ID='')['metrics'] for label,b in [('owner',owner),('moderator',mod),('member',member)]}
   target=member.rpc('State',ID='')['network']['id'];owner.rpc('CommunityAction',ID=group,Kind='member.remove',Body={'target':target})
   reject(member,'SyncCommunity',ID=group);view=member.rpc('CommunityState',ID=group,Channel=channel,Before=0);assert view['community']['denied']
   reject(member,'CommunityAction',ID=group,Kind='message.text',Body={'channelId':channel,'text':'não autorizado'})
   if args.metrics:
    report={'version':'0.5.1','scenario':'3 production backend processes over '+('real Radmin adapter, same PC' if args.radmin else 'loopback')+'; no native GUI measurement','durationSeconds':time.monotonic()-started,'ownerRestartedDuringScenario':True,'idleOwnerBeforeCommunities':idle,'afterScenario':metrics,'messageCommitRpcRoundtripMs':roundtrip,'manualPingMs':diagnostics.get('rttMicros',diagnostics['rttMillis']*1000)/1000,'renderCommitMs':None,'renderReason':'instrumented in UI, not measured by this API-only test','packetLoss':None,'packetLossReason':'TCP counters do not measure UDP loss'}
    pathlib.Path(args.metrics).write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8')
   print('PASS: three real desktop services, communities, owner-signed invites, roles, channel text, sync, durable outbox, restart, deduplication, revocation and preserved direct chat; '+('real Radmin adapter on one PC' if args.radmin else 'loopback'))
  finally:
   for b in processes:b.close()
if __name__=='__main__':main()
