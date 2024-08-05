# Users Service

A service to handle authentication and authorization


## Build (bin, docker)

See Makefile's -build- commands


## Deploy (k8s)

See Makefile's -deploy- command

Note: check the deploy yaml files and set the required secrets and env vars


## Databases

<b>PostgreSQL<b/>

`database: auth`

`user: dinghy-auth`

Contains tokens, credentials, permissions

See Makefile's -db- commands to run migration and access db (use .envrc for the connection string)

<b>Redis (In Progress)<b> 
