





## Application components
| Container | Description |
| --- | ----------- |
| **vmwhiz-caddy** | Entry point of the application. Proxies all requests matching `/api/*` to `vmwhiz-backend` and the rest to `vmwhiz-frontend`. Employs HTTPS by default. |
| **vmwhiz-frontend** | Vue3 frontend application |
| **vmwhiz-backend** | Go backend. Handles all requests matching `/api/*`. Note: in development the backend uses the [Air](https://github.com/air-verse/air) utility for hot-reloading the backend when there are changes to the source code. |
| **vmwhiz-db** | Postgres database for the backend |


## Using dev environment
1. Change `POSTGRES_PASSWORD` in [.db.env](/.db.env)
2. (Optional) Modify values inside [.env](.env)
3. Bring up the stack\
`cd docker && docker compose up`\
You should now be able to navigate to https://localhost and access the frontend UI.

## Environment variables
1. [.db.env](/.db.env) - Database variables\
Backend and database containers read from the same file
2. [.env](.env) - General purpose environment variables\
The file should look something like this:
```env
VMWHIZ_SCHEME=https
VMWHIZ_HOSTNAME=localhost
VMWHIZ_PORT=443
```
- `VMHIZ_SCHEME`: Enables/disables HTTPS. If, for some reason, you want to disable https, you have to also comment out a server directive in the [Caddyfile](/docker/Caddyfile) aswell.
- `VMWHIZ_HOSTNAME`: The hostname of the machine. Caddy will automatically provision SSL certificates for the specified hostname (unless https is disabled).
- `VMWHIZ_PORT`: The port on which Caddy listens for incoming requests.

All the above variables are read both by Caddy and by the frontend for generating the base URL of the instance.