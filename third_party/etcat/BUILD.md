# Compilação do auxiliar privado

Fontes e Cargo.lock correspondentes acompanham o setup em EasyTier-corresponding-source.zip. Abra esse arquivo para modificar e recompilar. Nenhum contrato restringe engenharia reversa para depurar modificações LGPL.

Ambiente usado: Rust/Cargo 1.95.0 x86_64-pc-windows-gnu; LLVM MinGW 20261020 msvcrt x86_64; Go 1.27.3. Ferramentas locais, sem instalação de drivers. O target GNU requer libgcc.a, libgcc_eh.a e libpthread.a do diretório self-contained do Rust; coloque somente esses três arquivos numa pasta gcc-runtime-libs (não misture todo o CRT do Rust com LLVM).

1. Extraia o pacote preservando a estrutura work/etcat-source/etcat-ccb45cd84c20b2aa1e6615a3f62a83827e961695 e work/easytier-minimal.
2. Compile work/protodesc: `go build -o protoc.exe .`. Esse compilador de descritores usa Buf protocompile, não requer protoc nativo. Defina PROTOC para o caminho absoluto desse executável.
3. Defina RUSTFLAGS para `-L native=CAMINHO/gcc-runtime-libs`; CC_x86_64_pc_windows_gnu e CARGO_TARGET_X86_64_PC_WINDOWS_GNU_LINKER para x86_64-w64-mingw32-clang.exe; AR_x86_64_pc_windows_gnu para llvm-ar.exe. Inclua LLVM/bin e Cargo/bin no PATH apenas do terminal.
4. Dentro da pasta etcat, crie .cargo/config.toml com as substituições do arquivo vendor-config.toml, ajustando directory para o caminho absoluto de work/vendor. Use `cargo build --release --locked --offline`.
5. Copie target/release/etcat.exe como build/bin/easytier/etcat-p2p.exe. Execute scripts/audit-pe.py. Atualize internal/overlay/manifest.json com o SHA-256 do seu binário e recompile o aplicativo. Hashes não são assinatura nem proteção contra um administrador local.

Os patches neste diretório mostram mudanças sobre os commits originais. Cargo.lock contém atualizações compatíveis de dependências; os avisos remanescentes são descritos no relatório do beta. Nunca reutilize o pacote de easytier-core que exigia Packet.dll. Não habilite features tun/faketcp/upnp. As fontes vendorizadas e suas licenças estão no arquivo correspondente, incluindo dependências de compilação. Não há promessa de hash binário idêntico em outra máquina/caminho/toolchain.
