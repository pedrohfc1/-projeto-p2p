// This independent module is deliberately not imported by the desktop application.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/identity"
	"projetop2p/internal/roomprobe"
	"tailscale.com/client/local"
	"tailscale.com/envknob"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

type tailStack interface {
	Dial(context.Context, string, string) (net.Conn, error)
}
type tailDialer struct{ stack tailStack }

func (d tailDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, _, e := net.SplitHostPort(address)
	if e != nil {
		return nil, e
	}
	ip, e := netip.ParseAddr(host)
	if e != nil || !tailIP(ip) {
		return nil, errors.New("laboratório aceita somente endereço IP Tailscale, sem fallback público/LAN")
	}
	return d.stack.Dial(ctx, network, address)
}
func tailIP(ip netip.Addr) bool {
	return netip.MustParsePrefix("100.64.0.0/10").Contains(ip) || netip.MustParsePrefix("fd7a:115c:a1e0::/48").Contains(ip)
}

// Noise must confirm the exact out-of-band authorized application Peer ID.
type gate struct{ allowed peer.ID }

func (g gate) InterceptPeerDial(p peer.ID) bool                 { return p == g.allowed }
func (g gate) InterceptAddrDial(p peer.ID, _ ma.Multiaddr) bool { return p == g.allowed }
func (g gate) InterceptAccept(network.ConnMultiaddrs) bool      { return true }
func (g gate) InterceptSecured(_ network.Direction, p peer.ID, _ network.ConnMultiaddrs) bool {
	return p == g.allowed
}
func (g gate) InterceptUpgraded(c network.Conn) (bool, control.DisconnectReason) {
	return c.RemotePeer() == g.allowed, 0
}

func authURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Hostname() == "login.tailscale.com" && u.Port() == "" && u.User == nil
}
func openBrowser(raw string) error {
	if !authURL(raw) {
		return errors.New("URL de autenticação fora do domínio oficial")
	}
	if runtime.GOOS == "windows" {
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", raw).Start()
	}
	return exec.Command("xdg-open", raw).Start()
}
func pathOf(p *ipnstate.PingResult) string {
	if p == nil {
		return "desconhecido"
	}
	if p.Err != "" {
		return "inacessível"
	}
	if p.PeerRelay != "" {
		return "peer-relay"
	}
	if p.DERPRegionID != 0 {
		return "DERP"
	}
	if p.Endpoint != "" {
		return "direto confirmado por ping disco"
	}
	return "desconhecido"
}
func stateDir(dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) == filepath.VolumeName(dir)+string(os.PathSeparator) {
		return errors.New("use uma pasta absoluta exclusiva do laboratório")
	}
	if info, e := os.Lstat(dir); e == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("pasta não pode ser link")
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return e
	}
	marker := filepath.Join(dir, ".peetwopee-tsnet-lab")
	if _, e = os.Stat(marker); os.IsNotExist(e) && len(entries) > 0 {
		return errors.New("a pasta contém dados que não pertencem ao laboratório")
	}
	if e = protectDirectory(dir); e != nil {
		return e
	}
	return os.WriteFile(marker, []byte("private laboratory state; do not export"), 0600)
}

// A userspace tsnet listener does not bind an OS socket. This bounded loopback
// bridge adapts the listener only; Noise still runs end-to-end inside its bytes.
func bridge(ctx context.Context, listener net.Listener, target string) func() {
	ctx, cancelBridge := context.WithCancel(ctx)
	var wg sync.WaitGroup
	slots := make(chan struct{}, 8)
	stop := context.AfterFunc(ctx, func() { listener.Close() })
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			incoming, e := listener.Accept()
			if e != nil {
				return
			}
			select {
			case slots <- struct{}{}:
			default:
				incoming.Close()
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-slots }()
				defer incoming.Close()
				out, e := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", target)
				if e != nil {
					return
				}
				defer out.Close()
				cancel := context.AfterFunc(ctx, func() { incoming.Close(); out.Close() })
				defer cancel()
				incoming.SetDeadline(time.Now().Add(90 * time.Second))
				out.SetDeadline(time.Now().Add(90 * time.Second))
				done := make(chan struct{})
				go func() { io.Copy(out, incoming); out.Close(); close(done) }()
				io.Copy(incoming, out)
				incoming.Close()
				<-done
			}()
		}
	}()
	return func() { cancelBridge(); stop(); listener.Close(); wg.Wait() }
}
func authenticate(ctx context.Context, s *tsnet.Server, open bool) (*local.Client, *ipnstate.Status, error) {
	if e := s.Start(); e != nil {
		return nil, nil, e
	}
	client, e := s.LocalClient()
	if e != nil {
		return nil, nil, e
	}
	lastURL := ""
	requested := false
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		query, cancel := context.WithTimeout(ctx, 3*time.Second)
		state, e := client.Status(query)
		cancel()
		if e == nil {
			if state.BackendState == "Running" {
				return client, state, nil
			}
			if state.BackendState == "NeedsLogin" && !requested {
				requested = true
				_ = client.StartLoginInteractive(ctx)
			}
			if state.AuthURL != "" && state.AuthURL != lastURL {
				if !authURL(state.AuthURL) {
					return nil, nil, errors.New("URL de autenticação rejeitada")
				}
				lastURL = state.AuthURL
				if open {
					if e = openBrowser(state.AuthURL); e != nil {
						return nil, nil, e
					}
					fmt.Println("Autenticação aberta no navegador. Confirme sua tailnet; nenhuma chave está no executável.")
				} else {
					fmt.Println("Abra pessoalmente a URL oficial (não compartilhe este link):", state.AuthURL)
				}
			}
			if state.BackendState == "NeedsMachineAuth" {
				fmt.Println("O administrador da tailnet precisa aprovar este nó.")
			}
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func allowSingleLoadProbe(ctx context.Context, h host.Host, allowed peer.ID, deadline time.Time) {
	var used atomic.Bool
	valid := func() bool {
		return ctx.Err() == nil && time.Now().Before(deadline)
	}
	roomprobe.SetHandler(h, func(remote peer.ID) (func() bool, bool) {
		if remote != allowed || !valid() || !used.CompareAndSwap(false, true) {
			return nil, false
		}
		return valid, true
	})
}

func writeSummary(summary map[string]any, report string) error {
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	if report != "" {
		return os.WriteFile(report, raw, 0600)
	}
	return nil
}

func run() error {
	base, _ := os.UserConfigDir()
	dir := flag.String("state-dir", filepath.Join(base, "PeeTwoPee-tsnet-lab"), "estado exclusivo; nunca use o perfil principal")
	name := flag.String("name", "peetwopee-lab", "nome deste nó tsnet")
	allow := flag.String("allow", "", "Peer ID autorizado por canal externo")
	remote := flag.String("peer", "", "multiaddr Tailscale completo esperado")
	port := flag.Int("port", 7447, "porta somente dentro da tailnet")
	duration := flag.Int("duration", 0, "sonda explícita: 0 (somente escutar), 10, 30 ou 60 segundos")
	load := flag.Bool("load", false, "autorizar sonda de carga limitada com o peer indicado, usando o mesmo protocolo do app")
	open := flag.Bool("open-auth", false, "abrir navegador quando a autorização for necessária")
	report := flag.String("report", "", "gravar resumo numérico local em JSON")
	checkUserspace := flag.Bool("check-userspace", false, "inicializar SDK sem login e verificar ausência de TUN/driver carregado")
	identityOnly := flag.Bool("identity-only", false, "mostrar identidade persistente do laboratório sem iniciar Tailscale")
	flag.Parse()
	if *duration != 0 && *duration != 10 && *duration != 30 && *duration != 60 {
		return errors.New("duração deve ser 0,10,30,60")
	}
	if *load {
		switch {
		case *checkUserspace || *identityOnly:
			return errors.New("-load requer conexão à tailnet")
		case *remote == "" && (*duration != 0 || *allow == ""):
			return errors.New("para receber uma sonda de carga, use -load -allow PEER_ID")
		case *remote != "" && *duration == 0:
			return errors.New("para iniciar uma sonda de carga, use -load -peer MULTIADDR -duration 10, 30 ou 60")
		}
	}
	if *port < 1024 || *port > 65535 {
		return errors.New("porta inválida")
	}
	for _, key := range []string{"TS_AUTHKEY", "TS_AUTH_KEY", "TS_CLIENT_SECRET", "TSNET_FORCE_LOGIN", "TS_CLIENT_ID", "TS_ID_TOKEN", "TS_AUDIENCE", "TS_CONTROL_URL"} {
		if os.Getenv(key) != "" {
			return errors.New("remova variáveis de autenticação Tailscale do ambiente; este laboratório utiliza login interativo")
		}
	}
	if e := stateDir(*dir); e != nil {
		return e
	}
	key, e := identity.Load(filepath.Join(*dir, "ed25519"))
	if e != nil {
		return e
	}
	self, _ := peer.IDFromPrivateKey(key)
	fmt.Println("Peer ID do laboratório:", self)
	if *identityOnly {
		return nil
	}
	allowed := peer.ID("")
	if *allow != "" {
		allowed, e = peer.Decode(*allow)
		if e != nil {
			return e
		}
	}
	var target *peer.AddrInfo
	if *remote != "" {
		target, e = peer.AddrInfoFromString(*remote)
		if e != nil {
			return e
		}
		if allowed != "" && allowed != target.ID {
			return errors.New("Peer ID não corresponde ao autorizado")
		}
		allowed = target.ID
	}
	if allowed == "" && !*checkUserspace {
		return errors.New("informe -allow PEER_ID; use -identity-only para trocar identidades antes de conectar")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	server := &tsnet.Server{Dir: filepath.Join(*dir, "tailnet"), Hostname: *name, Logf: func(string, ...any) {}, UserLogf: func(string, ...any) {}}
	defer server.Close()
	envknob.Setenv("TS_DISABLE_PORTMAPPER", "true")
	envknob.SetNoLogsNoSupport()
	if *checkUserspace {
		if e := server.Start(); e != nil {
			return e
		}
		client, e := server.LocalClient()
		if e != nil {
			return e
		}
		query, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		status, e := client.Status(query)
		if e != nil {
			return e
		}
		result := map[string]any{"tun": status.TUN, "wintunLoaded": driverLoaded(), "portMapperDisabled": os.Getenv("TS_DISABLE_PORTMAPPER") == "true", "state": status.BackendState, "authenticated": status.BackendState == "Running"}
		raw, _ := json.Marshal(result)
		fmt.Println(string(raw))
		if status.TUN || driverLoaded() {
			return errors.New("TUN/driver inesperado")
		}
		return nil
	}
	authCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	client, status, e := authenticate(authCtx, server, *open)
	cancel()
	if e != nil {
		return fmt.Errorf("autenticação não concluída: %w", e)
	}
	if status.TUN {
		return errors.New("nó inesperadamente reportou TUN; laboratório interrompido")
	}
	fmt.Printf("Peers visíveis na tailnet: %d (visibilidade não comprova ACL/alcance do protocolo).\n", len(status.Peer))
	var ip netip.Addr
	for _, candidate := range status.TailscaleIPs {
		if candidate.Is4() {
			ip = candidate
			break
		}
	}
	if !ip.IsValid() {
		return errors.New("IPv4 Tailscale indisponível")
	}
	announcement, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, *port))
	dialer := tailDialer{newTailnetDialer(server, client)}
	host, e := libp2p.New(
		libp2p.Identity(key),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.NoTransports,
		libp2p.Transport(tcp.NewTCPTransport, tcp.WithDialerForAddr(func(ma.Multiaddr) (tcp.ContextDialer, error) {
			return dialer, nil
		})),
		libp2p.DisableRelay(),
		libp2p.Security(noise.ID, noise.New),
		libp2p.ConnectionGater(gate{allowed}),
		libp2p.AddrsFactory(func([]ma.Multiaddr) []ma.Multiaddr { return []ma.Multiaddr{announcement} }),
	)
	if e != nil {
		return e
	}
	defer host.Close()
	// Host.Addrs are intentionally replaced for invitation output; find the actual OS listener from Network.ListenAddresses.
	listeners := host.Network().ListenAddresses()
	if len(listeners) != 1 {
		return errors.New("listener local inesperado")
	}
	localPort, e := listeners[0].ValueForProtocol(ma.P_TCP)
	if e != nil {
		return e
	}
	tailListener, e := server.Listen("tcp", fmt.Sprintf(":%d", *port))
	if e != nil {
		return e
	}
	defer bridge(ctx, tailListener, net.JoinHostPort("127.0.0.1", localPort))()
	var grantExpires time.Time
	if *load && target == nil {
		grantExpires = time.Now().Add(90 * time.Second)
		allowSingleLoadProbe(ctx, host, allowed, grantExpires)
	}
	fmt.Printf("Endereço real: %s/p2p/%s\n", announcement, self)
	fmt.Println("Tailnet autenticada. Um convite PeeTwoPee NÃO concede acesso à tailnet. Nenhum transporte do app foi alterado.")
	if !grantExpires.IsZero() {
		fmt.Println("Uma sonda de carga autorizada para o peer indicado pelos próximos 90 segundos. Ctrl+C cancela.")
		timer := time.NewTimer(time.Until(grantExpires))
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
			fmt.Println("Autorização de carga expirada.")
		}
		return nil
	}
	if target == nil || *duration == 0 {
		<-ctx.Done()
		return nil
	}
	connect, cancel := context.WithTimeout(ctx, 15*time.Second)
	e = host.Connect(connect, *target)
	cancel()
	if errors.Is(e, errTailnetPeerNotVisible) {
		return errors.New("peer não está visível nesta tailnet: confirme o endereço e o acesso concedido pelo administrador")
	}
	if e != nil {
		return errors.New("peer não alcançável ou não autorizado; verifique tailnet, ACL e Peer ID (sem fallback)")
	}
	if *load {
		result, err := roomprobe.Run(ctx, host, target.ID, *duration, func() bool {
			return ctx.Err() == nil && host.Network().Connectedness(target.ID) == network.Connected
		})
		if err != nil {
			return fmt.Errorf("sonda de carga não concluída: %w", err)
		}
		return writeSummary(map[string]any{
			"at":           time.Now().UnixMilli(),
			"mode":         "tailscale-lab",
			"testType":     "limited-load",
			"seconds":      result.DurationSeconds,
			"samples":      result.TotalBlocks,
			"payloadBytes": result.PayloadBytes,
			"rttMean":      result.RTTMeanMs,
			"rttP95":       result.RTTP95Ms,
			"variationMs":  result.VariationMs,
			"kbps":         result.PayloadKbps,
			"conditions":   "sonda de carga limitada sobre libp2p em tsnet, com o mesmo protocolo e limites da sonda do app",
		}, *report)
	}
	samples := []float64{}
	paths := []string{}
	started := time.Now()
	for i := 0; i < *duration; i++ {
		sampleAt := time.Now()
		pctx, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
		result, received := <-ping.Ping(pctx, host, target.ID)
		cancel()
		if received && result.Error == nil {
			samples = append(samples, float64(result.RTT)/float64(time.Millisecond))
		}
		if i%5 == 0 {
			pctx, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
			for _, a := range target.Addrs {
				if raw, e := a.ValueForProtocol(ma.P_IP4); e == nil {
					if addr, e := netip.ParseAddr(raw); e == nil {
						p, e := client.Ping(pctx, addr, tailcfg.PingDisco)
						if e == nil {
							path := pathOf(p)
							paths = append(paths, path)
							fmt.Println("Caminho verificado:", path)
						}
						break
					}
				}
			}
			cancel()
		}
		if wait := time.Second - time.Since(sampleAt); wait > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	sort.Float64s(samples)
	var mean, p95 any
	if len(samples) > 0 {
		total := 0.0
		for _, v := range samples {
			total += v
		}
		mean = total / float64(len(samples))
		p95 = samples[(len(samples)*95+99)/100-1]
	}
	summary := map[string]any{"at": time.Now().UnixMilli(), "mode": "tailscale-lab", "seconds": time.Since(started).Seconds(), "samples": len(samples), "rttMean": mean, "rttP95": p95, "jitter": nil, "kbps": nil, "fps": nil, "interruptions": 0, "noRxSeconds": nil, "pathSamples": paths, "conditions": "sonda libp2p 32 bytes / 1 Hz, sobre tsnet; sem mídia/carga; ausência de resposta não comprova perda UDP"}
	return writeSummary(summary, *report)
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
