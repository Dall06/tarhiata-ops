# Planning: Features Extra / Futuras para Tarhiata-ops

Este documento recopila las tareas y funcionalidades que se han identificado como "Extras" o "Nice-to-have" para que el PaaS sea competitivo a largo plazo, pero que no bloquean el desarrollo core actual.

## 1. Estrategia de Backups Automáticos (Bases de Datos a S3)
**Estado:** `Pendiente`
- **Descripción:** Actualmente los volúmenes de las bases de datos (Postgres/Mongo) son persistentes en el disco del nodo. Si el nodo se corrompe físicamente o el proveedor Cloud sufre una caída severa, hay riesgo de pérdida total.
- **Acción requerida:** Implementar un mecanismo interno (como un Cronjob en contenedor Docker) que ejecute un *dump* de la base de datos y lo suba a un Bucket S3 (ej. AWS S3, Cloudflare R2 o Vultr Object Storage) de manera automática cada 24 horas.

## 2. Pipeline CI/CD Interno (Git Webhooks)
**Estado:** `Pendiente`
- **Descripción:** Tarhiata-ops permite el despliegue de contenedores, pero la actualización frente a nuevo código en repositorios requiere un re-despliegue manual.
- **Acción requerida:** Levantar un servicio interno de Webhooks en el Manager Node que exponga un endpoint. Este endpoint recibirá eventos push de GitHub/GitLab y ejecutará un `docker service update` automático sobre los contenedores afectados.

## 3. Seguridad Avanzada de Llaves SSH
**Estado:** `Pendiente`
- **Descripción:** Las llaves privadas `.pem` y `_rsa` autogeneradas para administrar los nodos (tanto el master como los workers de base de datos/logs) se almacenan en texto plano en el directorio local `~/.ssh/`.
- **Acción requerida:** Integrar un sistema de cifrado simétrico (AES) que cifre las llaves en disco. El CLI podría solicitar una *passphrase* maestra al arrancar la sesión, garantizando que un malware local no pueda comprometer la infraestructura Cloud.
