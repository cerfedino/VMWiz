<img width="100%" src="./resources/form.png" />





## Application components
| Container | Description |
| --- | ----------- |
| **vmwiz-caddy** | Entry point of the application. Proxies all requests matching `/api/*` to `vmwiz-backend` and the rest to `vmwiz-frontend`. Employs HTTPS by default. |
| **vmwiz-frontend** | Vue3 frontend application |
| **vmwiz-backend** | Go backend. Handles all requests matching `/api/*`. Note: The backend uses the [Air](https://github.com/air-verse/air) utility in development for hot-reloading. |
| **vmwiz-db** | Postgres database for the backend |
| **vmwiz-notifier** | [Apprise](https://github.com/caronc/apprise) service allowing us to send notifications to a multitude of services.


## Bringing up the stack
1. **Change `POSTGRES_PASSWORD` in [.db.env](/.db.env)**
2. (Optional) **Add a notification endpoint into [notifier_config.yml](/docker/notifier_config.yml) such that you also get notifications.**
3. **Modify PVE values inside [.backend.env](.backend.env)**
4. **Setup SSH**\
The backend estabilishes SSH sessions to the Cluster management (CM) node and the Compute node (CN). To that end, you need to supply valid SSH credentials.\
  **5.1 Cluster Management node and Compute node**\
  Populate [docker/ssh/cm_pkey.key](docker/ssh/cm_pkey.key) and [docker/ssh/comp_pkey.key](docker/ssh/comp_pkey.key) with valid private keys for the root user.
  Add the CM and Comp host fingerprints to the [docker/ssh/known_hosts](docker/ssh/known_hosts) file. Finally, adjust the related environment variables in [.backend.env](.backend.env).\
  **5.2 Default VM credentials**\
  Each VM created through VMWiz will both accept the public key supplied by the requesting student/organization and another "universal" public key shared by every VM.
  Populate [docker/ssh/vm_univ_pubkey.pkey](docker/ssh/vm_univ_pubkey.pkey) and [docker/ssh/vm_univ_privkey.key](docker/ssh/vm_univ_privkey.pkey) with a valid key pair.
5. **Modify Netcenter values inside of [.backend.env](.backend.env)**\
The backend uses the Netcenter HTTP API, which requires authentication. To that end, insert the credentials of a valid user. 
6. **Modify Keycloak values inside of [.backend.env](.backend.env)**\
The backend has an OpenID client to authenticate SOSETH users. To that end, adjust the Keycloak-related environment variables in [.backend.env](.backend.env).
7. **Bring up the stack**\
`cd docker && docker compose up`\
You should now be able to navigate to https://localhost and access the frontend UI.

## Environment variables
1. [.db.env](/.db.env) - Database variables
2. [.env](.env) - General purpose environment variables\
The file should look something like this:
```env
VMWIZ_SCHEME=https
VMWIZ_HOSTNAME=localhost
VMWIZ_PORT=443
```
- `VMWIZ_HOSTNAME`: The hostname of the machine. Caddy will automatically provision SSL certificates for the specified hostname (unless https is disabled).
- `VMWIZ_PORT`: The port on which Caddy listens for incoming requests.

All the above variables are read both by Caddy and by the frontend for generating the base URL of the instance.

3. [.backend.env](.backend.env) - Backend-specific environment variables\
The file should look something like this:
```env
# PVE API authentication
# https://pve.proxmox.com/wiki/Proxmox_VE_API#API_Tokens
PVE_HOST=https://manage.vsos.ethz.ch
PVE_USER=root@pam
PVE_TOKENID=
PVE_UUID=

# Cluster manager SSH
SSH_CM_HOST=cm-lee.sos.ethz.ch 
SSH_CM_USER=root
## Leave empty if your private key has no passphrase
SSH_CM_PKEY_PASSPHRASE=

# Comp node SSH
SSH_COMP_HOST=comp-epyc-lee-3.sos.ethz.ch
SSH_COMP_USER=root
## Leave empty if your private key has no passphrase
SSH_COMP_PKEY_PASSPHRASE=


NETCENTER_HOST=https://www.netcenter.ethz.ch
NETCENTER_USER=sys-sos-vm-service
NETCENTER_PWD=


KEYCLOAK_ISSUER_URL="https://auth.sos.ethz.ch/auth/realms/master"
KEYCLOAK_CLIENT_ID="vmwiz-dev"
KEYCLOAK_CLIENT_SECRET=""
```

The PVE variables should be set according to a valid PVE API key of the form `<PVE_USER>!<PVE_TOKENID>=<PVE_UUID>`
