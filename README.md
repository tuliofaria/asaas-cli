# asaas-cli

CLI em Go para a [API v3 do Asaas](https://docs.asaas.com), inspirada no `gh`, feita para ser consumida por agentes de IA e automações: saída sempre em JSON, sem cores, sem prompts.

## Como rodar

### Requisitos

- [Go](https://go.dev/dl/) 1.25 ou mais recente (`go version` para conferir)
- `make` (opcional; já vem no macOS e na maioria das distribuições Linux)
- Uma chave de API do Asaas, gerada no painel em Integrações → Chaves de API. Para testar sem mexer em dinheiro de verdade, crie uma conta em [sandbox.asaas.com](https://sandbox.asaas.com).

> **Ainda não tem conta no Asaas?** Crie pelo [meu link de parceiro](https://www.asaas.com/parceiros/devpleno) e pague tarifas promocionais, menores que as da tabela padrão: Pix R$ 0,99, boleto R$ 1,89 e cartão 2,93% + R$ 0,29. É um link de indicação. Valores conferidos em outubro de 2026 em [devpleno.com/parceiros](https://devpleno.com/parceiros), onde ficam sempre atualizados.

### Instalação

```bash
git clone https://github.com/tuliofaria/asaas-cli.git
cd asaas-cli
make install        # compila e instala em ~/.local/bin/asaas
```

`~/.local/bin` precisa estar no `PATH`. Se `asaas` não for encontrado depois da instalação, adicione ao `~/.zshrc` (ou `~/.bashrc`):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Outras formas:

```bash
make install PREFIX=/usr/local   # instala em /usr/local/bin/asaas
make build                       # só compila em ./bin/asaas, sem instalar
go run . --help                  # roda direto do código, sem compilar
make uninstall                   # remove o binário instalado
```

Com `go install github.com/tuliofaria/asaas-cli@latest` o binário fica em `$(go env GOPATH)/bin` com o nome `asaas-cli`, não `asaas`.

### Primeiros passos

```bash
# 1. Cadastre a conta. Use aspas simples: a chave começa com "$aact_" e o shell
#    trataria "$aact" como variável.
asaas auth add --name sandbox-dev --key '$aact_hmlg_...' --sandbox

#    Para não deixar a chave no histórico do shell, passe pelo stdin:
pbpaste | asaas auth add --name sandbox-dev --key - --sandbox

# 2. Confira se a chave funciona
asaas auth status --verify

# 3. Faça a primeira chamada
asaas customer list --limit 5
```

Sem arquivo de configuração, dá para usar só a variável de ambiente:

```bash
ASAAS_API_KEY='$aact_hmlg_...' asaas customer list --limit 5
```

Todo comando tem `--help` com a descrição completa, o tipo de cada flag e quais são `[required]`.

### Testes

```bash
make test           # go test -race ./...
```

Os testes usam um servidor HTTP falso local: não precisam de chave nem de rede.

## Contrato de saída

| Situação | Destino | Formato |
|---|---|---|
| Sucesso | stdout | JSON indentado (corpo cru da resposta do Asaas) |
| Erro | stderr | `{"error": {"type": "...", "message": "...", ...}}` |

Tipos de erro e exit codes: `usage_error` (2), `auth_error`/`config_error` (3), `network_error` (4), `api_error`/`response_error`/`internal_error` (1). `api_error` inclui `status_code` e o array `errors` do Asaas.

## Autenticação

Ordem de resolução:

1. `--account <nome>`: conta específica do arquivo, só nesta execução.
2. `ASAAS_API_KEY`: usada direto, ignorando o arquivo.
3. `active_account` de `~/.config/asaas/config.json`.

Ambiente: `--sandbox`/`--production` sempre vencem. Sem flag, vale o `environment` da conta; com `ASAAS_API_KEY`, valem `ASAAS_ENV`, depois o prefixo da chave (`$aact_hmlg_` = sandbox, `$aact_prod_` = produção) e, por último, produção.

```bash
asaas auth add --name sandbox-dev --key "$KEY" --sandbox
printf '%s' "$KEY" | asaas auth add --name producao-firma --key - --production
asaas auth switch --name sandbox-dev
asaas auth status --verify
asaas auth list
asaas auth remove --name producao-firma
```

O arquivo é criado com permissão `0600` (diretório `0700`). Variáveis de ambiente: `ASAAS_API_KEY`, `ASAAS_ENV`, `ASAAS_CONFIG_DIR`, `ASAAS_BASE_URL`.

## Comandos

```bash
asaas customer list --name Maria --limit 20
asaas customer list --all                      # percorre todas as páginas
asaas customer get cus_000005219613
asaas customer create --name "Ana" --cpf-cnpj 24971563792 --email ana@x.com
asaas customer update cus_000005219613 --mobile-phone 47999376637
asaas customer delete cus_000005219613
asaas customer restore cus_000005219613

# Notas fiscais de serviço (NFS-e)
asaas fiscal-info municipal-options            # o que a prefeitura exige
asaas fiscal-info save --email fiscal@empresa.com --simples-nacional=true --municipal-inscription 123456
asaas fiscal-info services --description desenvolvimento   # municipalServiceId
asaas invoice create --payment pay_080225913252 --service-description "Consultoria" \
  --observations "Mensal" --value 300 --effective-date 2026-10-31 --municipal-service-id 60380 \
  --retain-iss=false --iss 2 --pis 0 --cofins 0 --csll 0 --inss 0 --ir 0
asaas invoice authorize inv_000000000232       # emite agora, sem esperar effectiveDate
asaas invoice list --status AUTHORIZED --effective-date-from 2026-10-01 --all
asaas invoice update inv_000000000232 --iss 3.5
asaas invoice cancel inv_000000000232

# Qualquer endpoint sem comando dedicado:
asaas api /payments --query status=PENDING
asaas api /payments -X POST -f customer=cus_000005219613 -f billingType=PIX -F value=129.90 -f dueDate=2026-12-31
```

Toda a documentação (tipos de flag, `[required]`, formato de saída) está no `--help` de cada comando.

## Usando com agentes de IA

A CLI foi feita para ser operada por agentes como Claude Code, Codex e Cursor: o agente descobre os comandos pelo `--help` e lê as respostas em JSON. Basta instalar o `asaas`, cadastrar uma conta e dar ao agente instruções como estas. Elas podem ir no `CLAUDE.md`, no `AGENTS.md` ou direto no chat:

```text
Você tem acesso à CLI `asaas` para operar minha conta no Asaas (API v3).

- Antes de usar um comando, leia `asaas --help` e `asaas <comando> --help`.
  O help mostra o tipo de cada flag e quais são [required].
- Sucesso sai em JSON no stdout. Erro sai em JSON no stderr, com exit code
  diferente de 0. Use jq para filtrar.
- Em listagens, use --all quando precisar de todos os registros.
- Para endpoints sem comando dedicado, use `asaas api <caminho>`.
- Comandos de leitura (list, get, auth status) podem rodar à vontade.
- Antes de qualquer comando que cria, altera, cancela ou emite algo (create,
  update, delete, authorize, cancel, `asaas api -X POST/PUT/DELETE`), me
  mostre exatamente o que vai rodar e espere minha confirmação. Emitir ou
  cancelar nota fiscal não tem volta.
- Na dúvida sobre a conta ou o ambiente, rode `asaas auth status`. Para
  testar, use --sandbox.
```

Exemplos de pedidos:

```text
Liste as notas fiscais com status ERROR com data de emissão em setembro e
outubro de 2026, agrupe pelo código de erro devolvido pela prefeitura e me
diga quantas são e o valor total de cada grupo. Não altere nada.
```

```text
Quais cobranças do cliente cus_000005219613 foram pagas no cartão pelo
Asaas e quais foram baixadas manualmente como recebidas em dinheiro?
```

## Estrutura

```
main.go                 entrypoint (os.Exit(cmd.Execute(version)))
cmd/                    árvore Cobra: root, version
  auth/                 add, switch, status, list, remove
  customer/             list, get, create, update, delete, restore
  invoice/              list, get, create, update, authorize, cancel (NFS-e)
  fiscalinfo/           get, save, municipal-options, services, nbs-codes, federal-service-codes, national-portal
  api/                  requisição crua para qualquer endpoint
internal/config/        config.json via Viper, variáveis ASAAS_*, resolução de credenciais
internal/cmdutil/       Factory (estilo gh), saída JSON, erros tipados, campos declarativos
pkg/asaasclient/        cliente HTTP genérico (headers, erros, paginação)
```

### Adicionando um recurso novo (ex.: `payment`)

1. Crie `cmd/payment/` com um grupo `cmdutil.NewGroup(...)`.
2. Declare filtros e campos como `[]cmdutil.Field` (parâmetro da API, flag, tipo, descrição, obrigatório). Parâmetros com ponto (`taxes.iss`) viram objetos aninhados; `Enum` e o tipo `Date` são validados antes da requisição.
3. Use `cmdutil.RunList` para listagens e `cmdutil.RunRequest` para get/create/update/delete.
4. Registre o grupo em `cmd/root.go`.

O teste `TestEveryCommandIsDocumented` falha se algum comando ficar sem `Short`/`Long` ou se alguma flag não documentar o tipo.

## Como este projeto nasceu

O esqueleto da CLI foi gerado por um agente de IA a partir do prompt abaixo. Os comandos de notas fiscais e o restante vieram depois, em pedidos menores na mesma conversa. Serve de ponto de partida se você quiser fazer algo parecido para outra API.

<details>
<summary>Prompt inicial (<code>/goal</code>)</summary>

~~~~text
/goal Objetivo: Criar uma CLI profissional em Go (Golang) inspirada na estrutura do gh (GitHub CLI) para interagir com a API do Asaas v3. A CLI será consumida por agentes de IA e automações locais. O projeto deve ser arquitetado para cobrir toda a API do Asaas através de uma estrutura altamente escalável.

🛠️ Requisitos Técnicos e de Arquitetura

1. Frameworks: Utilizar Cobra CLI para os comandos e Viper para gerenciar as configurações de arquivos e variáveis de ambiente.
2. Hierarquia de Autenticação e Multi-Contas:
	• Prioridade 1 (Variável de Ambiente): Verificar se existe a variável ASAAS_API_KEY. Se existir, utilizá-la imediatamente (ignora o arquivo de configuração). A flag global --sandbox define o ambiente correspondente.
	• Prioridade 2 (Arquivo na Home): Se a variável de ambiente não estiver definida, ler o arquivo de configuração localizado em ~/.config/asaas/config.json.
	• Estrutura do Mapa de Contas (config.json): O arquivo deve armazenar um mapa de contas identificadas por um apelido (nome), registrando para cada uma a Chave de API e se pertence ao ambiente Sandbox ou Produção. Deve haver uma chave active_account para indicar qual está em uso. Exemplo de estrutura desejada:
	```json
	{
	  "active_account": "sandbox-dev",
	  "accounts": {
	    "sandbox-dev": { "api_key": "asaas_key_1...", "environment": "sandbox" },
	    "producao-firma": { "api_key": "asaas_key_2...", "environment": "production" }
	  }
	}
	```
3. Gerenciamento de Ambiente em Tempo de Execução: Criar uma flag global --sandbox (booleana) para forçar o uso do ambiente de testes (https://asaas.com) ou --production para produção, independentemente do padrão da conta ativa.
4. Arquitetura Escalável (Foco em Cobrir Toda a API):
	• Criar um pacote interno de cliente HTTP genérico (ex: pkg/asaasclient).
	• Esse cliente deve ser instanciado dinamicamente com base na conta ativa ou na variável de ambiente injetada. Deve gerenciar os headers globais (access_token, Content-Type) e tratar erros de rede.
5. Formato de Saída (Foco em IA):
	• Toda saída bem-sucedida deve ser enviada para o stdout estritamente em JSON estruturado e formatado (json.MarshalIndent). Desativar cores ou prompts interativos.
	• Erros de rede ou respostas HTTP de erro devem ir para o stderr em formato JSON.

🔍 Auto-Descoberta por IA via --help

• Todos os comandos e subcomandos devem conter descrições Short e Long detalhadas.
• Todas as flags do Cobra devem documentar o tipo de dado esperado de forma exata.
• Parâmetros obrigatórios devem usar cmd.MarkFlagRequired(), garantindo que apareçam como [required] no --help.

📁 Estrutura de Comandos e Contexto de Autenticação

O agente deve estruturar a CLI incluindo um subcomando de autenticação:
• asaas auth add --name "sandbox-dev" --key "sua_chave" --sandbox -> Adiciona/conecta uma nova conta ao mapa na Home do usuário.
• asaas auth switch --name "sandbox-dev" -> Altera a conta ativa no arquivo JSON.
• asaas auth status -> Retorna em JSON os dados da conta que está conectada no momento (ocultando parte da chave por segurança).
• asaas customer [list|create|get|update|delete] -> Módulos normais da API do Asaas que consumirão a conta ativa.

🚀 O que o agente deve entregar no primeiro passo:

1. Arquivos main.go, go.mod e estrutura idiomática de pastas (/cmd, /pkg/asaasclient).
2. Configuração do Viper para criar e ler a pasta ~/.config/asaas/ de forma nativa e multiplataforma.
3. Implementação completa do grupo de comandos asaas auth [add|switch|status].
4. Implementação do comando base de uma entidade (asaas customer list) já validando o fluxo: se resolve a autenticação via ENV ou via arquivo JSON da Home, efetuando o disparo do HTTP real.
~~~~

</details>

O prompt dá `https://asaas.com` como endereço do sandbox, mas o certo é `https://api-sandbox.asaas.com/v3`. A CLI já usa o endereço correto.

## Licença

[MIT](LICENSE)
