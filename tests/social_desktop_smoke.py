import pathlib,sys,tempfile,time
from webui_smoke import Bridge
binary=pathlib.Path(sys.argv[1]).resolve();parent=pathlib.Path('data').resolve();parent.mkdir(exist_ok=True)
def until(fn,seconds=25):
 deadline=time.monotonic()+seconds
 while True:
  result=fn()
  if result:return result
  if time.monotonic()>deadline:raise AssertionError('activity convergence timeout')
  time.sleep(.25)
with tempfile.TemporaryDirectory(prefix='social-integration-',dir=parent) as tmp:
 root=pathlib.Path(tmp);a=Bridge(binary,root/'a');b=Bridge(binary,root/'b');c=Bridge(binary,root/'c')
 try:
  aid=a.rpc('State',ID='')['network']['id'];bid=b.rpc('State',ID='')['network']['id']
  a.rpc('AddContact',Text=b.rpc('Invite'));b.rpc('AddContact',Text=a.rpc('Invite'))
  a.rpc('Send',ID=bid,Text='original activity fixture');message=until(lambda:next((m for m in b.rpc('State',ID=aid)['messages'] if m['text']=='original activity fixture'),None));mid=message['id'];ra={'peer':bid};rb={'peer':aid}
  def action(peer,room,kind,target,text='',emoji='',enabled=False):return peer.rpc('ChatAction',Room=room,KindText=kind,Target=target,TextValue=text,Emoji=emoji,Enabled=enabled)
  try:action(b,rb,'edit',mid,'forged')
  except AssertionError:pass
  else:raise AssertionError('non-author edit accepted')
  b.close();action(a,ra,'edit',mid,'queued while recipient offline');b=Bridge(binary,root/'b')
  until(lambda:b.rpc('ActivityView',Room=rb,IDs=[aid+':'+mid],Replies=True)['overlays'].get(aid+':'+mid,{}).get('text')=='queued while recipient offline')
  reply=action(b,rb,'reply',mid,'authenticated reply');until(lambda:a.rpc('ActivityView',Room=ra,IDs=[aid+':'+mid],Replies=True)['overlays'].get(bid+':'+reply,{}).get('replyTo')==mid)
  action(b,rb,'reaction',mid,emoji='👍',enabled=True);until(lambda:len(a.rpc('ActivityView',Room=ra,IDs=[aid+':'+mid],Replies=True)['reactions'].get(aid+':'+mid,[]))==1)
  group=a.rpc('CreateCommunity',Name='Activity group fixture',Description='',Icon='');ticket=a.rpc('CommunityInvitePolicy',ID=group,Options={'kind':'multi','maxUses':5,'lifetimeHours':24});b.rpc('JoinCommunity',Text=ticket);c.rpc('JoinCommunity',Text=ticket)
  for peer in [b,c]:peer.rpc('SyncCommunity',ID=group)
  room={'group':group,'channel':'geral'};receipt=a.rpc('CommunityAction',ID=group,Kind='message.text',Body={'channelId':'geral','text':'community base'});target=receipt['id']
  until(lambda:next((m for m in b.rpc('CommunityState',ID=group,Channel='geral',Before=0)['messages'] if m['id']==target),None))
  action(a,room,'edit',target,'community edited');until(lambda:b.rpc('ActivityView',Room=room,IDs=[target],Replies=True)['overlays'].get(target,{}).get('text')=='community edited')
  action(c,room,'reaction',target,emoji='🎉',enabled=True);until(lambda:a.rpc('ActivityView',Room=room,IDs=[target],Replies=True)['reactions'].get(target,[]))
  action(a,room,'delete',target);until(lambda:c.rpc('ActivityView',Room=room,IDs=[target],Replies=True)['overlays'].get(target,{}).get('deleted'))
  action(a,ra,'delete',mid);until(lambda:b.rpc('ActivityView',Room=rb,IDs=[aid+':'+mid],Replies=True)['overlays'].get(aid+':'+mid,{}).get('deleted'))
  a.rpc('Send',ID=bid,Text='chat preserved after activities');until(lambda:any(m['text']=='chat preserved after activities' for m in b.rpc('State',ID=aid)['messages']))
  print('PASS: real integrated three-process signed activities, own-only edit/delete, reactions/replies, offline queue/restart recovery, tombstones and preserved direct/community chat; same PC')
 finally:
  for p in [a,b,c]:p.close()
