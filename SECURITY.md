# Segurança

O Projeto P2P 0.5.0 é experimental e não passou por auditoria independente. As verificações automatizadas ajudam a encontrar problemas, mas não são garantia de segurança. Não há compromisso de suporte a versões anteriores; ao publicar correções, use a versão mais recente e consulte o changelog. Veja a [auditoria 0.5](docs/SEGURANCA-v0.5.md): comunidades ainda não têm E2EE de grupo e o dono confirma as operações enquanto está online.

## Relatar uma vulnerabilidade

Quando o repositório real estiver configurado, o mantenedor deverá habilitar **Settings → Security → Private vulnerability reporting**. Use então **Security → Advisories → Report a vulnerability** para comunicação privada. Até esse canal estar habilitado, não publique detalhes exploráveis em issues; solicite ao mantenedor um canal privado, sem anexar segredos. Não há e-mail de segurança inventado neste projeto.

Informe versão, plataforma, passos mínimos e impacto com dados sintéticos. Nunca envie identity.key, banco de histórico, senha de VPN, token ou perfil completo. Aguarde coordenação de correção/divulgação com o mantenedor.

## Limites relevantes

Peer ID é identidade criptográfica, não identidade civil. Transporte e mensagens são autenticados, mas o histórico local não é criptografado em repouso. Processos com acesso ao mesmo usuário Windows podem acessar DPAPI; permissões Linux não protegem contra o próprio usuário/root. Não há proteção contra todas as formas de DoS, Sybil ou eclipse. Relays e fornecedores VPN observam metadados; DHT pode expor presença/endereço.

Use convites de pessoas confiáveis; conectar a endereços recebidos permite conexões à rede interna. A aplicação não configura VPNs nem escreve regras de firewall. O diagnóstico consultivo de regras não substitui o teste de conexão autenticada e ping. Não desative proteções para executar o programa.

Executável e instalador ainda não possuem certificado de assinatura de distribuição. O bootstrapper WebView2 é um componente Microsoft separado cuja assinatura é verificada durante o build. Confira checksums pelo canal oficial da release; hash isolado não autentica o publicador.

## Verificações de desenvolvimento

Auditoria de arquivos candidatos Git, gitleaks, `go mod verify`, `govulncheck`, `npm audit --audit-level=high`, testes e Actions fixadas por SHA estão na CI. Não publique uma release com verificações falhando. Dependabot propõe atualizações para revisão. Os testes com redes/PCs externos e aceitação do instalador são separados da CI normal.

## Aviso conhecido da versão 0.4.0

GO-2024-3218 afeta go-libp2p-kad-dht, sem versão corrigida conhecida na base Go. Pode comprometer disponibilidade/descoberta da DHT nativa. O modo VPN não inicia DHT, mas isso não elimina o aviso da distribuição. O gate de segurança bloqueia publicação enquanto não houver solução ou política de exceção explicitamente revisada. Fonte: https://pkg.go.dev/vuln/GO-2024-3218 .
