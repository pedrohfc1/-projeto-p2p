# 0004 — Registro genérico de provedores VPN

Hamachi e Radmin compartilham enumeração Windows somente leitura (GetAdaptersAddresses e metadados de driver), seleção de interface e ciclo de vida do host. internal/adapters/vpn.go centraliza IDs, nomes, prefixos esperados e evidências de driver. Nome amigável pode ser alterado; não é prova. Faixas 25/8 e 26/8 são pistas do perfil e não prova de associação ou de conectividade. Interfaces personalizadas podem ser escolhidas manualmente.

Cada VPN é independente: ausência de Hamachi não impede Radmin e vice-versa. Automático conserva libp2p nativo. A seleção explícita vincula escuta, origem TCP e anúncio ao IPv4 escolhido, sem DHT público, NAT mapping, AutoRelay ou DCUtR nesse host. Convites têm escopo assinado; contatos e histórico não são apagados ao mudar o modo. Endereços virtuais registrados são excluídos do anúncio nativo.

Monitoramento interrompe o host ao perder a interface e reabre quando possível, mantendo identidade. Adaptador ativo é estado local; conectividade exige handshake libp2p e ping real. Não usamos APIs de configuração nem um SDK Radmin/Hamachi. Não há scanner de membros ou exigência de VPNs simultâneas. Segurança e limites libp2p existentes continuam aplicados.
