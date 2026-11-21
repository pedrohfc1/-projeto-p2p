"""Create a local reproducible Wails permission-policy build (no module-cache edits)."""
import pathlib,subprocess,json,hashlib,shutil
root=pathlib.Path(__file__).resolve().parent.parent
version=subprocess.check_output(['go','list','-m','-f','{{.Version}}','github.com/wailsapp/wails/v2'],cwd=root,text=True).strip();assert version=='v2.17.0','Review new Wails version before building'
module=pathlib.Path(subprocess.check_output(['go','list','-m','-f','{{.Dir}}','github.com/wailsapp/wails/v2'],cwd=root,text=True).strip())
source=module/'internal/frontend/desktop/windows/frontend.go'
raw=source.read_text(encoding='utf-8');assert hashlib.sha256(raw.encode()).hexdigest()=='9598c7f21779e2332db788a9e09f3b4856d563ab5fc2c93bf7035743b9d7befa','Review changed Wails frontend source';old='chromium.SetGlobalPermission(edge.CoreWebView2PermissionStateAllow)';assert raw.count(old)==1,'Review upstream Wails permission policy before building'
out=root/'build/permission-policy';out.mkdir(parents=True,exist_ok=True)
def copy_source(src,dst):
 target=pathlib.Path(dst)
 if target.exists():target.chmod(0o644)
 return shutil.copyfile(src,dst)
local=out/'wails';shutil.copytree(module,local,dirs_exist_ok=True,copy_function=copy_source,ignore=shutil.ignore_patterns('.git','node_modules'))
(local/'internal/frontend/desktop/windows/frontend.go').write_text(raw.replace(old,'chromium.SetGlobalPermission(edge.CoreWebView2PermissionStateDefault)'),encoding='utf-8')
(out/'provenance.json').write_text(json.dumps({'module':'github.com/wailsapp/wails/v2@v2.17.0','originalSHA256':hashlib.sha256(raw.encode()).hexdigest(),'policy':'WebView2 Default (user prompt), no automatic grants'}),encoding='utf-8')
print(local)
