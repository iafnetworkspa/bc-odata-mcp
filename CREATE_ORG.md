# Creazione Organizzazione GitHub

Per creare l'organizzazione `yamamotospa` su GitHub:

1. Vai su https://github.com/organizations/new
2. Scegli "Create an organization"
3. Inserisci:
   - **Organization name:** `yamamotospa`
   - **Contact email:** (la tua email)
   - **This organization belongs to:** (seleziona il tuo account)
4. Scegli il piano (Free è sufficiente per iniziare)
5. Completa la configurazione

Una volta creata l'organizzazione, puoi creare il repository eseguendo:

```bash
gh repo create yamamotospa/bc-odata-mcp --public --source=. --remote=origin --description "MCP Server for Microsoft Business Central OData API - Enables LLM and Cursor integration with BC APIs"
```

Oppure puoi creare il repository direttamente dall'interfaccia web di GitHub e poi fare push del codice.

