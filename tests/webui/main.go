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

	"projetop2p/internal/community"
	"projetop2p/internal/desktop"
	"projetop2p/internal/easytierlab"
	"projetop2p/internal/invite"
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
			LabConfig   easytierlab.Config
			Method      string
			ID          string
			Text        string
			Settings    localdb.Settings
			Channel     string
			Before      uint64
			Kind        string
			Body        community.Body
			Name        string
			Description string
			Icon        string
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		var result any
		var err error
		switch req.Method {
		case "StartIntegratedLab":
			err = core.StartIntegratedLab(req.LabConfig)
		case "StopIntegratedLab":
			core.StopIntegratedLab()
		case "IntegratedLabState":
			result = core.IntegratedLabState()
		case "ProbeIntegratedLab":
			result, err = core.ProbeIntegratedLab(req.ID, req.Text)
		case "CreateCommunity":
			result, err = core.CreateCommunity(req.Name, req.Description, req.Icon)
		case "CommunityState":
			result, err = core.CommunityState(req.ID, req.Channel, req.Before)
		case "CommunityAction":
			result, err = core.CommunityAction(req.ID, req.Kind, req.Body)
		case "CommunityInvite":
			result, err = core.CommunityInvite(req.ID)
		case "JoinCommunity":
			result, err = core.JoinCommunity(req.Text)
		case "SyncCommunity":
			err = core.SyncCommunity(req.ID)
		case "DiscardCommunityPending":
			err = core.DiscardCommunityPending(req.ID, req.Text)
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
		case "LegacyInvite":
			if strings.HasPrefix(req.Text, community.InvitePrefix) {
				result, err = community.LegacyInvite(req.Text)
			} else {
				result, err = invite.Legacy(req.Text)
			}
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
		script := `<script nonce="` + token + `">const rpc=async(method,args={})=>{const r=await fetch('/rpc',{method:'POST',headers:{'Content-Type':'application/json','X-QA-Token':'` + token + `'},body:JSON.stringify({Method:method,...args})});const v=await r.json();if(!r.ok)throw Error(v.error);return v};window.go={main:{App:{StartIntegratedLab:config=>rpc('StartIntegratedLab',{LabConfig:config}),StopIntegratedLab:()=>rpc('StopIntegratedLab'),IntegratedLabState:()=>rpc('IntegratedLabState'),ProbeIntegratedLab:(raw,text)=>rpc('ProbeIntegratedLab',{ID:raw,Text:text}),CreateCommunity:(name,description,icon)=>rpc('CreateCommunity',{Name:name,Description:description,Icon:icon}),CommunityState:(id,channel,before)=>rpc('CommunityState',{ID:id,Channel:channel,Before:before}),CommunityAction:(id,kind,body)=>rpc('CommunityAction',{ID:id,Kind:kind,Body:body}),CommunityInvite:id=>rpc('CommunityInvite',{ID:id}),JoinCommunity:text=>rpc('JoinCommunity',{Text:text}),SyncCommunity:id=>rpc('SyncCommunity',{ID:id}),DiscardCommunityPending:(id,op)=>rpc('DiscardCommunityPending',{ID:id,Text:op}),State:id=>rpc('State',{ID:id}),Send:(id,text)=>rpc('Send',{ID:id,Text:text}),AddContact:text=>rpc('AddContact',{Text:text}),ConnectInvite:text=>rpc('ConnectInvite',{Text:text}),TestConnectivity:id=>rpc('TestConnectivity',{ID:id}),SaveSettings:settings=>rpc('SaveSettings',{Settings:settings}),LegacyInvite:text=>rpc('LegacyInvite',{Text:text}),Invite:()=>rpc('Invite'),Copy:text=>navigator.clipboard.writeText(text)}}};</script>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(strings.Replace(strings.Replace(string(b), "script-src 'self'", "script-src 'self' 'nonce-"+token+"'", 1), "</head>", script+"</head>", 1)))
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
