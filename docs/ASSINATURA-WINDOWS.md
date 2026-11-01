# Windows: confiança, diagnóstico e assinatura

O build de testes 0.5.0 não tem assinatura Authenticode. Nenhum certificado de assinatura foi fornecido, importado ou acessado. O bootstrapper WebView2 embutido é separado e tem assinatura Microsoft verificada no empacotamento; isso não assina nosso app/setup.

SmartScreen avalia reputação e pode avisar sobre software desconhecido; ausência de assinatura é verificável separadamente. Detecção pelo Defender exige identificar ameaça e arquivo/hash. Não há evidência suficiente para chamar o aviso do usuário de falso positivo. A consulta local ao Defender ficou indisponível. [Microsoft SmartScreen](https://learn.microsoft.com/en-us/windows/security/operating-system-security/virus-and-threat-protection/microsoft-defender-smartscreen/).

Com o arquivo original e sem mudar proteções:

```powershell
./scripts/inspect-windows.ps1 -Files @('build/bin/ProjetoP2P.exe','build/bin/ProjetoP2P-Setup-v0.5.0.exe')
Get-FileHash ./build/bin/ProjetoP2P-Setup-v0.5.0.exe -Algorithm SHA256
```

O script apenas consulta hashes, assinatura e detecções já registradas para esses caminhos. Não faz varredura completa, não altera proteção e não garante ausência de malware. `unavailable` significa consulta indisponível. Em detecção real ou HashMismatch, interromper distribuição e investigar. Conservar mensagem exata, hash e nome da ameaça; eventual envio à Microsoft requer autorização específica. Não solicitar exceção de antivírus ou desativar SmartScreen.

## Assinatura preparada, ainda não executada

É necessário certificado legítimo de Code Signing, chave privada acessível ao operador e Windows SDK SignTool. O titular/publicador deve ser real; não fabricamos nome verificado. Provisionamento do certificado ocorre fora do repositório. O script exige thumbprint explícito, certificado válido, chave privada e EKU de assinatura de código no store CurrentUser/My.

```powershell
./scripts/desktop.ps1 -Installer -SigningThumbprint 'THUMBPRINT_REAL_DE_40_HEXADECIMAIS'
./scripts/inspect-windows.ps1 -Files @('build/bin/ProjetoP2P.exe','build/bin/ProjetoP2P-Setup-v0.5.0.exe') -RequireSigned
```

Ordem: build do app → assina app → comprime payload → build/manifest/assinatura do desinstalador → build/manifest/assinatura do setup → verifica payload e assinaturas → hashes finais. O placeholder acima não é credencial e será recusado. Não modificar PE depois de assinar. O script usa SHA-256 e carimbo RFC3161 com `/tr http://timestamp.digicert.com /td SHA256`, depois verifica status e thumbprint. [SignTool Microsoft](https://learn.microsoft.com/en-us/windows/win32/seccrypto/signtool), [timestamp DigiCert](https://knowledge.digicert.com/solution/troubleshooting-timestamping-problems).

Somente análise sintática foi feita no caminho de assinatura: não há prova de assinatura bem-sucedida sem certificado. Assinatura válida identifica publicador/integridade, não garante reputação imediata ou ausência de detecção. Builds sem assinatura podem ser comparados byte a byte; timestamps e assinaturas fazem o artefato assinado variar, por isso comparar também o build unsigned de referência.

## GitHub Actions e Store

CI existente mantém Actions por SHA, permissões mínimas e testes Windows/Linux; somente o job publicador de Release tem contents:write. Execução manual gera artefatos sem publicar. Tag deve coincidir com VERSION e segurança precisa passar; GO-2024-3218 ainda bloqueia. Nenhuma nova tag/release foi criada.

O parâmetro de assinatura está pronto para uso por um operador ou runner protegido com certificado provisionado. **Não foi adicionado job que importa PFX/segredos automaticamente.** Para automatizar depois: ambiente protegido com aprovação, acesso somente a tag revisada, isolamento do runner/token, identidade OIDC/HSM se disponível e limpeza garantida. Nunca disponibilizar certificado/segredo a PR de fork ou armazenar PFX/senha no Git. Fixar e revisar a ferramenta do provedor escolhido antes da integração.

Microsoft Store/MSIX é opção futura: conta de publicador verificada, identidade de pacote e publisher coerentes, políticas de rede/privacidade, pacote e atualização testados. Não temos credenciais, reserva de nome ou submissão, e o nome final do produto ainda não está definido. Não converter ou publicar silenciosamente o instalador atual.
