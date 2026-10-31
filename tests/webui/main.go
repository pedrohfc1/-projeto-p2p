// Development-only visual QA harness. Production desktop uses Wails IPC.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"projetop2p/internal/desktop"
	"projetop2p/internal/localdb"
)

func main() {
	dir := flag.String("data", "", "Required isolated test profile")
	port := flag.Int("port", 8792, "Loopback port")
	flag.Parse()
	if *dir == "" {
		panic("--data required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	core, e := desktop.Open(ctx, *dir)
	if e != nil {
		panic(e)
	}
	defer core.Close()
	nonce := make([]byte, 32)
	if _, e = rand.Read(nonce); e != nil {
		panic(e)
	}
	token := hex.EncodeToString(nonce)
	origin := fmt.Sprintf("http://127.0.0.1:%d", *port)
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("X-QA-Token") != token || r.Header.Get("Origin") != origin {
			http.Error(w, "forbidden", 403)
			return
		}
		var req struct {
			Method   string
			ID       string
			Text     string
			Settings localdb.Settings
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		var result any
		var err error
		switch req.Method {
		case "State":
			result, err = core.State(req.ID)
		case "Send":
			err = core.Send(req.ID, req.Text)
		case "AddContact":
			err = core.AddContact(req.Text)
		case "ConnectInvite":
			result, err = core.ConnectInvite(req.Text)
		case "TestConnectivity":
			result, err = core.TestConnectivity(req.ID)
		case "Invite":
			result, err = core.Invite()
		case "SaveSettings":
			err = core.SaveSettings(req.Settings)
		default:
			http.Error(w, "unknown method", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(400)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	})
	fs := http.FileServer(http.Dir("frontend/dist"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != strings.TrimPrefix(origin, "http://") {
			http.Error(w, "invalid host", 403)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path != "/" {
			fs.ServeHTTP(w, r)
			return
		}
		b, e := os.ReadFile("frontend/dist/index.html")
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		script := `<script>const rpc=async(method,args={})=>{const r=await fetch('/rpc',{method:'POST',headers:{'Content-Type':'application/json','X-QA-Token':'` + token + `'},body:JSON.stringify({Method:method,...args})});const v=await r.json();if(!r.ok)throw Error(v.error);return v};window.go={main:{App:{State:id=>rpc('State',{ID:id}),Send:(id,text)=>rpc('Send',{ID:id,Text:text}),AddContact:text=>rpc('AddContact',{Text:text}),ConnectInvite:text=>rpc('ConnectInvite',{Text:text}),TestConnectivity:id=>rpc('TestConnectivity',{ID:id}),SaveSettings:settings=>rpc('SaveSettings',{Settings:settings}),Invite:()=>rpc('Invite'),Copy:text=>navigator.clipboard.writeText(text)}}};</script>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(strings.Replace(string(b), "<head>", "<head>"+script, 1)))
	})
	listener, e := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if e != nil {
		panic(e)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(c)
	}()
	fmt.Println("QA only:", origin)
	if e = server.Serve(listener); e != nil && e != http.ErrServerClosed {
		panic(e)
	}
}
