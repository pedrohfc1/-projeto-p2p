"""Public-source gates tested with isolated synthetic Git repositories."""
import pathlib,unittest,tempfile,subprocess,shutil,sys
ROOT=pathlib.Path(__file__).resolve().parent.parent
class DistributionGates(unittest.TestCase):
 def test_valid_tag_and_reject_mismatched_tag(self):
  v=(ROOT/'VERSION').read_text().strip();parts=v.split('.');parts[-1]=str(int(parts[-1])+1);other='.'.join(parts)
  for tag,ok in [('v'+v,True),('v'+other,False),('v'+v+'-rc.1',False)]:
   r=subprocess.run([sys.executable,str(ROOT/'scripts/check-version.py'),tag],capture_output=True)
   self.assertEqual(r.returncode==0,ok)
 def audit_fixture(self,name,content,ignored=False):
  (ROOT/'dist').mkdir(exist_ok=True)
  with tempfile.TemporaryDirectory(dir=ROOT/'dist') as folder:
   p=pathlib.Path(folder);(p/'scripts').mkdir();shutil.copyfile(ROOT/'scripts/audit-public.py',p/'scripts/audit-public.py')
   subprocess.run(['git','init','-q'],cwd=p,check=True)
   if ignored:(p/'.gitignore').write_text(name+'\n')
   f=p/name;f.parent.mkdir(parents=True,exist_ok=True);f.write_text(content)
   if ignored:subprocess.run(['git','add','-f',name],cwd=p,check=True)
   return subprocess.run([sys.executable,str(p/'scripts/audit-public.py')],cwd=p,capture_output=True,text=True)
 def test_forced_staged_private_file_is_blocked(self):
  r=self.audit_fixture('.env','PASSWORD=synthetic-value',True)
  self.assertNotEqual(r.returncode,0);self.assertNotIn('synthetic-value',r.stdout)
 def test_credential_under_innocent_name_is_blocked_without_echo(self):
  value='gh'+'p_'+'a'*36
  r=self.audit_fixture('notes.md',value)
  self.assertNotEqual(r.returncode,0);self.assertNotIn(value,r.stdout)
 def test_normal_source_passes(self):
  self.assertEqual(self.audit_fixture('example.go','package example').returncode,0)
if __name__=='__main__':unittest.main()
