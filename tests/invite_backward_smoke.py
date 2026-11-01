"""Run unchanged 0.5.0 helper and current helper with isolated profiles."""
import pathlib,sys,tempfile,socket,time
from webui_smoke import Bridge
old,new=map(lambda s:pathlib.Path(s).resolve(),sys.argv[1:3])
def configure(b,name):
 a=next(a for a in b.rpc('State',ID='')['connectivity']['adapters'] if a['up'] and '127.0.0.1' in a['ipv4'])
 with socket.socket() as s:s.bind(('127.0.0.1',0));port=s.getsockname()[1]
 b.rpc('SaveSettings',Settings={'name':name,'mode':'interface','interfaceId':a['id'],'port':port,'bootstrap':[],'relays':[],'assisted':False})
parent=pathlib.Path('data').resolve();parent.mkdir(exist_ok=True)
with tempfile.TemporaryDirectory(prefix='p2p-compat-',dir=parent) as folder:
 nodes=[]
 try:
  a=Bridge(old,pathlib.Path(folder)/'old');nodes.append(a);b=Bridge(new,pathlib.Path(folder)/'new');nodes.append(b)
  configure(a,'Antigo');configure(b,'Novo')
  ai=a.rpc('Invite');bi=b.rpc('Invite');assert b.rpc('ConnectInvite',Text=ai)['connected'];assert a.rpc('ConnectInvite',Text=b.rpc('LegacyInvite',Text=bi))['connected']
  aid=a.rpc('State',ID='')['network']['id'];bid=b.rpc('State',ID='')['network']['id']
  b.rpc('Send',ID=aid,Text='Novo para antigo');a.rpc('Send',ID=bid,Text='Antigo para novo')
  g=a.rpc('CreateCommunity',Name='Compatibilidade',Description='',Icon='');r=b.rpc('JoinCommunity',Text=a.rpc('CommunityInvite',ID=g));assert r['status']=='participante',r
  b.rpc('CommunityAction',ID=g,Kind='message.text',Body={'channelId':'geral','text':'Texto novo, sequenciador antigo'})
  v=a.rpc('CommunityState',ID=g,Channel='geral',Before=0);assert any(m['text']=='Texto novo, sequenciador antigo' for m in v['messages'])
  g2=b.rpc('CreateCommunity',Name='Dono novo',Description='',Icon='');compact=b.rpc('CommunityInvite',ID=g2);r=a.rpc('JoinCommunity',Text=b.rpc('LegacyInvite',Text=compact));assert r['status']=='participante',r
  print('PASS: unchanged 0.5.0 + current contact invites, bidirectional DM, old owner/new member and new owner/old member')
 finally:
  for n in nodes:n.close()
