"""Fresh integrated default and exact selection persistence: real isolated backend/helper, same PC."""
import pathlib,sys,tempfile
from webui_smoke import Bridge
binary=pathlib.Path(sys.argv[1]).resolve();parent=pathlib.Path('data').resolve();parent.mkdir(exist_ok=True)
with tempfile.TemporaryDirectory(prefix='network-default-',dir=parent) as temp:
 profile=pathlib.Path(temp)/'profile';b=Bridge(binary,profile)
 try:
  s=b.rpc('State',ID='');assert s['settings']['mode']=='integrated';assert s['integrated']['running'];assert s['network']['addresses'];identity=s['network']['id']
  group=b.rpc('CreateCommunity',Name='Persistent network fixture',Description='',Icon='')
  b.rpc('SelectNetwork',ID='auto',Text='');assert b.rpc('State',ID='')['settings']['mode']=='auto'
 finally:b.close()
 b=Bridge(binary,profile)
 try:
  s=b.rpc('State',ID='');assert s['settings']['mode']=='auto' and s['network']['id']==identity
  try:b.rpc('SelectNetwork',ID='interface',Text='fixture-adapter-does-not-exist')
  except AssertionError:pass
  else:raise AssertionError('Missing interface unexpectedly available')
  s=b.rpc('State',ID='');assert s['settings']['mode']=='interface' and s['settings']['interfaceId']=='fixture-adapter-does-not-exist';assert s['networkError'];assert s['network']['id']==identity
 finally:b.close()
 b=Bridge(binary,profile)
 try:
  s=b.rpc('State',ID='');assert s['settings']['mode']=='interface' and s['network']['id']==identity
  b.rpc('SelectNetwork',ID='integrated',Text='');s=b.rpc('State',ID='');assert s['settings']['mode']=='integrated' and s['integrated']['running'];assert s['network']['id']==identity;assert any(c['id']==group for c in s['communities'])
 finally:b.close()
print('PASS: first start integrated/helper running, explicit auto restored, unavailable manual choice persisted without fallback, identity/community preserved on return to integrated; same PC')
