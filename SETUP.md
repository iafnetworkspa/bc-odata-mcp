# Setup Instructions

## Prerequisiti

- Go 1.21 o superiore
- Credenziali Business Central (Client ID, Client Secret, Tenant ID)
- Accesso alle API OData di Business Central

## Configurazione Rapida

### Opzione 1: Usando PowerShell (Windows)

1. Copia il file di esempio:
   ```powershell
   Copy-Item setup-bc-env.ps1.example setup-bc-env.ps1
   ```

2. Modifica `setup-bc-env.ps1` con le tue credenziali reali

3. Esegui lo script per impostare le variabili d'ambiente:
   ```powershell
   . .\setup-bc-env.ps1
   ```

4. Compila e avvia il server:
   ```powershell
   go build -o bc-odata-mcp.exe ./cmd/server
   .\bc-odata-mcp.exe
   ```

### Opzione 2: Usando variabili d'ambiente (Cross-platform)

1. Copia il file di esempio:
   ```bash
   cp config.example.env .env
   ```

2. Modifica `.env` con le tue credenziali reali

3. Carica le variabili d'ambiente:
   - **Windows (PowerShell):**
     ```powershell
     Get-Content .env | ForEach-Object {
       if ($_ -match '^([^=]+)=(.*)$') {
         [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
       }
     }
     ```
   - **Linux/Mac:**
     ```bash
     export $(cat .env | xargs)
     ```

4. Compila e avvia il server:
   ```bash
   go build -o bc-odata-mcp ./cmd/server
   ./bc-odata-mcp
   ```

## Configurazione in Cursor

1. Apri le impostazioni di Cursor (File > Preferences > Settings)
2. Cerca "MCP Servers" o modifica direttamente il file di configurazione
3. Aggiungi la configurazione seguendo l'esempio in `cursor-mcp-config.json.example`
4. Assicurati di sostituire:
   - Il percorso del comando con il percorso reale del tuo eseguibile
   - Tutte le variabili d'ambiente con i valori reali

## Note Importanti

⚠️ **Sicurezza:**
- I file `setup-bc-env.ps1` e `.env` sono già nel `.gitignore` e NON verranno committati
- Non condividere mai le credenziali in chiaro
- Usa variabili d'ambiente o un sistema di gestione segreti in produzione

## Troubleshooting

### Errore di autenticazione (401)
- Verifica che le credenziali OAuth siano corrette
- Controlla che il `BC_SCOPE_API` sia corretto
- Assicurati che il `BC_TOKEN_URL` contenga il `TENANT_ID` corretto

### Errore di connessione
- Verifica che `BC_BASE_PATH` sia corretto
- Controlla che `BC_TENANT_ID`, `BC_ENVIRONMENT`, e `BC_COMPANY` siano corretti
- Verifica la connettività di rete

### Rate limiting (429)
Il server gestisce automaticamente il rate limiting con retry esponenziali. Se continui a ricevere errori:
- Aumenta i delay tra le richieste
- Riduci la frequenza delle query
- Usa la paginazione invece di query multiple

