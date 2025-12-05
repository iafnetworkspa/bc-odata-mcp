# Business Central OData MCP Server

Un server MCP (Model Context Protocol) in Go che espone tutte le API OData di Microsoft Business Central per l'utilizzo con LLM e Cursor.

## Caratteristiche

- ✅ Autenticazione OAuth 2.0 con gestione automatica del token
- ✅ Supporto completo per query OData (filter, select, orderby, top, skip)
- ✅ Paginazione automatica opzionale
- ✅ Gestione automatica dei retry e rate limiting
- ✅ Tools MCP per query generiche e operazioni specifiche
- ✅ Compatibile con Cursor e altri client MCP

## Prerequisiti

- Go 1.21 o superiore
- Credenziali Business Central (Client ID, Client Secret, Tenant ID)
- Accesso alle API OData di Business Central

## Installazione

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
go build -o bc-odata-mcp ./cmd/server
```

## Configurazione

1. Copia il file di esempio della configurazione:
```bash
cp config.example.env .env
```

2. Modifica `.env` con le tue credenziali Business Central:
```env
BC_CLIENT_ID=your_client_id_here
BC_CLIENT_SECRET=your_client_secret_here
BC_SCOPE_API=https://api.businesscentral.dynamics.com/.default
BC_TOKEN_URL=https://login.microsoftonline.com/{TENANT_ID}/oauth2/v2.0/token
BC_BASE_PATH=https://api.businesscentral.dynamics.com/v2.0/{TENANT_ID}/{ENVIRONMENT}/api/v2.0/companies({COMPANY_ID})/
BC_TENANT_ID=your_tenant_id_here
BC_ENVIRONMENT=Production
BC_COMPANY=your_company_id_here
```

**Nota:** Sostituisci `{TENANT_ID}`, `{ENVIRONMENT}`, e `{COMPANY_ID}` con i valori corretti nel `BC_TOKEN_URL` e `BC_BASE_PATH`.

## Utilizzo

### Esecuzione diretta

Il server MCP comunica tramite stdin/stdout usando JSON-RPC:

```bash
./bc-odata-mcp
```

### Configurazione in Cursor

1. Apri le impostazioni di Cursor
2. Vai alla sezione MCP Servers
3. Aggiungi la seguente configurazione:

```json
{
  "mcpServers": {
    "bc-odata": {
      "command": "/path/to/bc-odata-mcp",
      "env": {
        "BC_CLIENT_ID": "your_client_id",
        "BC_CLIENT_SECRET": "your_client_secret",
        "BC_SCOPE_API": "https://api.businesscentral.dynamics.com/.default",
        "BC_TOKEN_URL": "https://login.microsoftonline.com/{TENANT_ID}/oauth2/v2.0/token",
        "BC_BASE_PATH": "https://api.businesscentral.dynamics.com/v2.0/{TENANT_ID}/{ENVIRONMENT}/api/v2.0/companies({COMPANY_ID})/",
        "BC_TENANT_ID": "your_tenant_id",
        "BC_ENVIRONMENT": "Production",
        "BC_COMPANY": "your_company_id",
        "BC_API_TIMEOUT": "90"
      }
    }
  }
}
```

## Tools Disponibili

### `bc_odata_query`

Esegue una query OData generica contro le API di Business Central.

**Parametri:**
- `endpoint` (richiesto): Path dell'endpoint OData (es. 'ODV_List', 'BI_Invoices', 'Customers')
- `filter` (opzionale): Espressione OData $filter (es. "No eq '12345'")
- `select` (opzionale): Espressione OData $select per specificare i campi da restituire
- `orderby` (opzionale): Espressione OData $orderby (es. 'Document_Date desc')
- `top` (opzionale): Limite del numero di risultati
- `skip` (opzionale): Numero di risultati da saltare
- `paginate` (opzionale): Se true, recupera automaticamente tutte le pagine (default: false)

**Esempio:**
```json
{
  "name": "bc_odata_query",
  "arguments": {
    "endpoint": "BI_Invoices",
    "filter": "Order_No eq '12345'",
    "select": "No,Order_No,Amount,Document_Date",
    "orderby": "Document_Date desc",
    "top": 10
  }
}
```

### `bc_odata_get_entity`

Recupera un'entità specifica tramite la sua chiave.

**Parametri:**
- `endpoint` (richiesto): Path dell'endpoint OData
- `key` (richiesto): Valore della chiave dell'entità (es. numero ordine, numero fattura)

**Esempio:**
```json
{
  "name": "bc_odata_get_entity",
  "arguments": {
    "endpoint": "ODV_List",
    "key": "25ODV-VI-000291"
  }
}
```

### `bc_odata_count`

Ottiene il conteggio delle entità che corrispondono a un filtro.

**Parametri:**
- `endpoint` (richiesto): Path dell'endpoint OData
- `filter` (opzionale): Espressione OData $filter

**Esempio:**
```json
{
  "name": "bc_odata_count",
  "arguments": {
    "endpoint": "BI_Invoices",
    "filter": "Document_Date ge 2024-01-01"
  }
}
```

## Esempi di Utilizzo

### Query con filtro

Recupera tutte le fatture per un ordine specifico:
```json
{
  "name": "bc_odata_query",
  "arguments": {
    "endpoint": "BI_Invoices",
    "filter": "Order_No eq '25ODV-VI-000291'"
  }
}
```

### Query con paginazione

Recupera tutti gli ordini con paginazione automatica:
```json
{
  "name": "bc_odata_query",
  "arguments": {
    "endpoint": "ODV_List",
    "paginate": true,
    "orderby": "Document_Date desc"
  }
}
```

### Query con select e orderby

Recupera solo campi specifici ordinati per data:
```json
{
  "name": "bc_odata_query",
  "arguments": {
    "endpoint": "BI_Invoices",
    "select": "No,Order_No,Amount,Document_Date",
    "orderby": "Document_Date desc",
    "top": 50
  }
}
```

## Struttura del Progetto

```
bc-odata-mcp/
├── cmd/
│   └── server/
│       └── main.go          # Entry point del server
├── internal/
│   ├── bc/
│   │   ├── auth.go          # Gestione autenticazione OAuth 2.0
│   │   └── client.go        # Client OData con retry e paginazione
│   └── mcp/
│       ├── server.go        # Server MCP principale
│       └── types.go         # Tipi JSON-RPC e MCP
├── config.example.env        # File di configurazione di esempio
├── go.mod                    # Dipendenze Go
└── README.md                 # Questo file
```

## Sviluppo

### Test locale

Per testare il server localmente, puoi usare uno script di test che invia richieste JSON-RPC:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./bc-odata-mcp
```

### Build per produzione

```bash
go build -ldflags="-s -w" -o bc-odata-mcp ./cmd/server
```

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

## Licenza

Questo progetto è fornito "così com'è" per uso interno.

## Contributi

I contributi sono benvenuti! Per favore apri una issue o una pull request.

