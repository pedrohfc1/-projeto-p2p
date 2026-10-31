package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"projetop2p/internal/adapters"
	"projetop2p/internal/network"
)

type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(s string) error { *l = append(*l, s); return nil }
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var boot, relays, listens, announces list
	dir := flag.String("data", "data", "Diretório exclusivo desta identidade")
	relayServer := flag.Bool("relay-server", false, "Oferecer relay quando publicamente alcançável")
	private := flag.Bool("private", false, "Solicitar relay sem esperar AutoNAT (NAT/CGNAT)")
	public := flag.Bool("public", false, "Operador: declarar este nó publicamente alcançável")
	daemon := flag.Bool("daemon", false, "Executar sem ler stdin (operador de relay)")
	checkRelay := flag.String("check-relay", "", "Testar uma reserva Circuit Relay v2 com identidade temporária e encerrar")
	interfaces := flag.Bool("interfaces", false, "Listar adaptadores e evidências sem alterar o Windows")
	interfaceIP := flag.String("interface-ip", "", "IPv4 exclusivo, sem DHT/relay (diagnóstico VPN)")
	statusEvery := flag.Duration("status-every", 0, "Emitir diagnóstico JSON periódico (ex.: 30s); 0 desabilita")
	flag.Var(&boot, "bootstrap", "Multiendereço /p2p/ID; pode repetir")
	flag.Var(&relays, "relay", "Relay estático /p2p/ID; pode repetir")
	flag.Var(&listens, "listen", "Multiendereço de escuta; pode repetir")
	flag.Var(&announces, "announce", "Transporte público anunciado, ex.: /dns4/SEU_DOMINIO/tcp/4001, sem /p2p/ID; pode repetir")
	flag.Parse()
	if *interfaces {
		items, e := adapters.List()
		if e != nil {
			return e
		}
		return printJSON(items, true)
	}
	if *statusEvery < 0 {
		return fmt.Errorf("--status-every não pode ser negativo")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *checkRelay != "" {
		probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		result, e := network.CheckRelay(probeCtx, *checkRelay)
		if e != nil {
			return fmt.Errorf("teste de relay falhou: %w", e)
		}
		return printJSON(result, true)
	}
	if len(listens) == 0 {
		listens = list{"/ip4/0.0.0.0/tcp/0", "/ip6/::/tcp/0"}
		if *interfaceIP != "" {
			listens = list{fmt.Sprintf("/ip4/%s/tcp/4001", *interfaceIP)}
		}
	}
	n, err := network.New(ctx, network.Config{DataDir: *dir, Listen: listens, Bootstrap: boot, Relays: relays, RelayServer: *relayServer, ForcePrivate: *private, ForcePublic: *public, Announce: announces, LocalOnly: *interfaceIP != "", ScopedIP: *interfaceIP})
	if err != nil {
		return err
	}
	defer n.Close()
	fmt.Println("Peer ID:", n.Host.ID())
	for _, a := range n.Addresses() {
		fmt.Println("Endereço:", a)
	}
	if *relayServer {
		fmt.Println("Modo: Bootstrap DHT + Circuit Relay v2. --public declara alcance; não abre firewall nem cria IP público.")
	}
	fmt.Println("Comandos: /status | /peers | /addresses | /connect MULTIADDR | /send PEER_ID TEXTO | /quit")
	var statusTick <-chan time.Time
	if *statusEvery > 0 {
		ticker := time.NewTicker(*statusEvery)
		defer ticker.Stop()
		statusTick = ticker.C
	}
	lines := make(chan string)
	if !*daemon {
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 1024), 8192)
			for scanner.Scan() {
				select {
				case lines <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			close(lines)
		}()
	}
	previous := map[peer.ID]bool{}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-statusTick:
			if err = printJSON(n.Status(), false); err != nil {
				return err
			}
		case event := <-n.Events:
			fmt.Println(event)
		case m := <-n.Messages:
			fmt.Printf("Mensagem de %s: %q\n", m.From, m.Text)
		case <-tick.C:
			current := map[peer.ID]bool{}
			for _, p := range n.Host.Network().Peers() {
				current[p] = true
				if !previous[p] {
					fmt.Println("Conectado:", p)
				}
			}
			for p := range previous {
				if !current[p] {
					fmt.Println("Desconectado:", p)
				}
			}
			previous = current
		case line, ok := <-lines:
			if !ok {
				return nil
			}
			parts := strings.SplitN(strings.TrimSpace(line), " ", 3)
			opctx, cancel := context.WithTimeout(ctx, 8*time.Second)
			switch parts[0] {
			case "/status":
				if err = printJSON(n.Status(), true); err != nil {
					cancel()
					return err
				}
			case "/quit":
				cancel()
				return nil
			case "/peers":
				for _, p := range n.Host.Network().Peers() {
					fmt.Println(p)
				}
			case "/addresses":
				for _, a := range n.Addresses() {
					fmt.Println(a)
				}
			case "/connect":
				if len(parts) < 2 {
					fmt.Println("Uso: /connect MULTIADDR")
					break
				}
				p, e := network.ParseAddress(parts[1])
				if e == nil {
					e = n.Connect(opctx, p)
				}
				if e != nil {
					fmt.Println("Erro:", e)
				}
			case "/send":
				if len(parts) < 3 {
					fmt.Println("Uso: /send PEER_ID TEXTO")
					break
				}
				id, e := peer.Decode(parts[1])
				if e == nil {
					e = n.Send(opctx, id, parts[2])
				}
				if e != nil {
					fmt.Println("Erro:", e)
				} else {
					fmt.Println("Recebimento confirmado")
				}
			default:
				fmt.Println("Comando desconhecido")
			}
			cancel()
		}
	}
}

func printJSON(v any, pretty bool) error {
	e := json.NewEncoder(os.Stdout)
	if pretty {
		e.SetIndent("", "  ")
	}
	return e.Encode(v)
}
