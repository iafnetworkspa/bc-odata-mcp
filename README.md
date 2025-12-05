# Business Central OData MCP Server

Un server MCP (Model Context Protocol) in Go che espone tutte le API OData di Microsoft Business Central per l'utilizzo con LLM e Cursor.

## Caratteristiche

- ✅ Autenticazione OAuth 2.0 con gestione automatica del token
- ✅ Supporto completo per query OData (filter, select, orderby, top, skip)
- ✅ Paginazione automatica opzionale
- ✅ Gestione automatica dei retry e rate limiting
- ✅ Tools MCP per query generiche e operazioni specifiche
- ✅ Compatibile con Cursor e altri client MCP
- ✅ CI/CD automatico con GitHub Actions
- ✅ Semantic Versioning e Changelog automatico

## Prerequisiti

- Go 1.21 o superiore
- Credenziali Business Central (Client ID, Client Secret, Tenant ID)
- Accesso alle API OData di Business Central

## Installazione

### Da Source

1. Clona il repository:
```bash
git clone https://github.com/iafnetworkspa/bc-odata-mcp.git
cd bc-odata-mcp
```

2. Installa le dipendenze:
```bash
go mod download
```

3. Compila il server:
```bash
# Su Windows
go build -o bc-odata-mcp.exe ./cmd/server

# Su Linux/macOS
go build -o bc-odata-mcp ./cmd/server
```

Oppure usa il Makefile:
```bash
# Su Windows
make build-windows

# Su Linux/macOS
make build-linux  # o make build-darwin per macOS
```

### Da Release

Scarica l'ultima release dalla [pagina releases](https://github.com/iafnetworkspa/bc-odata-mcp/releases) e seleziona il binario appropriato per la tua piattaforma.

## Configurazione

1. Copia il file di esempio:
```bash
cp config.example.env .env
```

2. Modifica `.env` con le tue credenziali Business Central:
```env
BC_CLIENT_ID=your_client_id
BC_CLIENT_SECRET=your_client_secret
BC_TENANT_ID=your_tenant_id
BC_ENVIRONMENT=Production
BC_COMPANY=your_company
BC_BASE_PATH=https://api.businesscentral.dynamics.com/v2.0
BC_TOKEN_URL=https://login.microsoftonline.com/{TENANT_ID}/oauth2/v2.0/token
BC_SCOPE_API=https://api.businesscentral.dynamics.com/.default
```

3. Per Windows PowerShell, puoi anche usare lo script di setup:
```powershell
.\setup-bc-env.ps1.example
```

## Utilizzo

### Con Cursor

1. Configura il file MCP di Cursor (`~/.cursor/mcp.json` o `%USERPROFILE%\.cursor\mcp.json`):
```json
{
  "mcpServers": {
    "bc-odata": {
      "command": "C:\\path\\to\\bc-odata-mcp.exe",
      "env": {
        "BC_CLIENT_ID": "your_client_id",
        "BC_CLIENT_SECRET": "your_client_secret",
        "BC_TENANT_ID": "your_tenant_id",
        "BC_ENVIRONMENT": "Production",
        "BC_COMPANY": "your_company",
        "BC_BASE_PATH": "https://api.businesscentral.dynamics.com/v2.0",
        "BC_TOKEN_URL": "https://login.microsoftonline.com/{TENANT_ID}/oauth2/v2.0/token",
        "BC_SCOPE_API": "https://api.businesscentral.dynamics.com/.default"
      }
    }
  }
}
```

2. Riavvia Cursor per caricare il server MCP.

### Tools Disponibili

Il server espone i seguenti tools MCP:

#### `bc_odata_query`
Esegue una query OData generica.

**Parametri:**
- `endpoint` (string, required): Nome dell'endpoint OData (es. "ODV_List", "Customers")
- `filter` (string, optional): Filtro OData (es. "No eq '12345'")
- `select` (string, optional): Campi da selezionare (es. "No,Name,Amount")
- `orderby` (string, optional): Ordinamento (es. "Document_Date desc")
- `top` (number, optional): Limite risultati (es. 10)
- `skip` (number, optional): Numero di risultati da saltare
- `paginate` (boolean, optional): Se true, recupera tutte le pagine automaticamente

**Esempio:**
```json
{
  "endpoint": "ODV_List",
  "filter": "Document_Type eq 'Order'",
  "orderby": "Document_Date desc",
  "top": 10
}
```

#### `bc_odata_get_entity`
Recupera un'entità specifica per chiave.

**Parametri:**
- `endpoint` (string, required): Nome dell'endpoint OData
- `key` (string, required): Valore della chiave

**Esempio:**
```json
{
  "endpoint": "ODV_List",
  "key": "ORD-001"
}
```

#### `bc_odata_count`
Conta le entità che corrispondono a un filtro.

**Parametri:**
- `endpoint` (string, required): Nome dell'endpoint OData
- `filter` (string, optional): Filtro OData

**Esempio:**
```json
{
  "endpoint": "ODV_List",
  "filter": "Document_Type eq 'Order'"
}
```

#### `bc_odata_list_endpoints`
Elenca tutti gli endpoint OData disponibili in Business Central. Utile per scoprire entità e API disponibili.

**Parametri:**
- Nessuno

**Esempio:**
```json
{}
```

**Risposta:**
Restituisce un array di nomi di endpoint disponibili e il documento di servizio completo.

#### `bc_odata_get_metadata`
Ottiene i metadati OData per gli endpoint. Include la struttura delle entità, proprietà e relazioni.

**Parametri:**
- `endpoint` (string, optional): Nome dell'endpoint OData. Se omesso, restituisce tutti i metadati.

**Esempio:**
```json
{}
```

oppure per metadati specifici:
```json
{
  "endpoint": "ODV_List"
}
```

**Nota:** I metadati sono tipicamente in formato XML e contengono informazioni dettagliate su tutte le entità, proprietà, tipi di dati e relazioni disponibili nel servizio OData.

## Struttura del Progetto

```
bc-odata-mcp/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── bc/
│   │   ├── auth.go              # OAuth 2.0 authentication
│   │   └── client.go            # OData client
│   └── mcp/
│       ├── server.go             # MCP server implementation
│       ├── types.go              # MCP protocol types
│       └── server_test.go        # Tests
├── .github/
│   └── workflows/
│       ├── ci.yml                # CI workflow
│       ├── build.yml             # Build workflow
│       └── release.yml           # Release workflow
├── .gsemanticrelease.yml        # Semantic release config
├── CHANGELOG.md                 # Changelog (auto-generated)
├── config.example.env           # Example configuration
├── go.mod                       # Go dependencies
├── Makefile                     # Build automation
└── README.md                    # Questo file
```

## Sviluppo

### Test locale

Per testare il server localmente, puoi usare uno script di test che invia richieste JSON-RPC:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./bc-odata-mcp
```

### Build per produzione

```bash
# Su Windows
go build -ldflags="-s -w" -o bc-odata-mcp.exe ./cmd/server

# Su Linux/macOS
go build -ldflags="-s -w" -o bc-odata-mcp ./cmd/server
```

Oppure usa il Makefile:
```bash
make build-windows  # o make build-linux, make build-darwin
```

### Conventional Commits

Questo progetto usa [Conventional Commits](https://www.conventionalcommits.org/) per il versioning automatico. I commit devono seguire il formato:

- `feat:` per nuove funzionalità (minor version bump)
- `fix:` per bug fix (patch version bump)
- `BREAKING CHANGE:` o `!` per breaking changes (major version bump)
- `chore:`, `docs:`, `style:`, `refactor:`, `perf:`, `test:`, `build:`, `ci:` per altre modifiche

### CI/CD

Il progetto include workflow GitHub Actions per:
- **CI**: Build, test e lint su ogni push/PR
- **Build**: Build multi-piattaforma (Linux, Windows, macOS)
- **Release**: Versioning automatico e generazione changelog basati su Conventional Commits

## Sicurezza

- ⚠️ **Non committare mai** file `.env` o credenziali nel repository
- ✅ Usa variabili d'ambiente o un sistema di gestione segreti in produzione
- ✅ Il token OAuth viene cachato e rinnovato automaticamente
- ✅ Le comunicazioni con Business Central avvengono tramite HTTPS

## Troubleshooting

### Errore di autenticazione

Se ricevi errori 401, verifica:
- Le credenziali OAuth sono corrette
- Il `BC_SCOPE_API` è corretto
- Il `BC_TOKEN_URL` contiene il `TENANT_ID` corretto

### Errore di connessione

Se non riesci a connetterti a Business Central:
- Verifica che `BC_BASE_PATH` sia corretto
- Controlla che `BC_TENANT_ID`, `BC_ENVIRONMENT`, e `BC_COMPANY` siano corretti
- Verifica la connettività di rete

### Rate limiting

Il server gestisce automaticamente il rate limiting con retry esponenziali. Se continui a ricevere errori 429, considera di:
- Aumentare i delay tra le richieste
- Ridurre la frequenza delle query
- Usare la paginazione invece di query multiple

## Changelog

Vedi [CHANGELOG.md](CHANGELOG.md) per la lista completa delle modifiche.

## Licenza

Questo progetto è fornito "così com'è" per uso interno.

## Contributi

I contributi sono benvenuti! Per favore apri una issue o una pull request. Assicurati di seguire le [Conventional Commits](https://www.conventionalcommits.org/) per i messaggi di commit.
