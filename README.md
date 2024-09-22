





## Application components
| Container | Description |
| --- | ----------- |
| **vmwiz-caddy** | Entry point of the application. Proxies all requests matching `/api/*` to `vmwiz-backend` and the rest to `vmwiz-frontend`. Employs HTTPS by default. |
| **vmwiz-frontend** | Vue3 frontend application |
| **vmwiz-backend** | Go backend. Handles all requests matching `/api/*`. Note: The backend uses the [Air](https://github.com/air-verse/air) utility in development for for hot-reloading upon changes to the source code. |
| **vmwiz-db** | Postgres database for the backend |
| **vmwiz-notifier** | [caronc/Apprise](https://github.com/caronc/apprise) container allowing us to easily send notifications to a multitude of services.


## Bringing up the stack
1. **Change `POSTGRES_PASSWORD` in [.db.env](/.db.env)**
2. (Optional) **Modify values inside [.env](.env)**
3. (Optional) **Add a notification endpoint into [notifier_config.yml](/docker/notifier_config.yml) such that you also get notifications.**
4. **Modify PVE values inside [.pve.env](.pve.env)**
5. **Modify SSH values inside of [.pve.env](.pve.env) and setup SSH**\
The backend connects via SSH to the cluster management node (e.g `cm-lee.sos.ethz.ch`). To that end, you need to put your private key inside of [docker/ssh/pkey.key](docker/ssh/pkey.key). Make sure that your public key is in the `authorized_keys` file on the CM machine.
Add the CM host's fingerprint to the [docker/ssh/known_hosts](docker/ssh/known_hosts) file. Finally, adjust the environment variables inside of [.pve.env](.pve.env).
6. **Bring up the stack**\
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

3. [.pve.env](.pve.env) - PVE-related environment variables\
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
SSH_CM_USER=
# Leave empty if the private key has no passphrase
SSH_CM_PKEY_PASSPHRASE=
```

The PVE variables should be set according to a valid API key of the form `{PVE_USER}!{PVE_TOKENID}={PVE_UUID}`
