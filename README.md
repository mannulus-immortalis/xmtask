# XM Companies storage

### Overview

This repository contains example companies storage microservice.

### Requirements

* Docker is required for automated build and run.
* PostgreSQL and Kafka are required for manual run. Use **init.sql** to initialize DB.

### Configuration

All configuration is done with ENV variables. Local `.env` files are supported with [godotenv](https://github.com/joho/godotenv/)

Default configuration values are stored in `docker-compose.yaml`.

* **LISTEN_ADDRESS** - API entry point, example: ":8081"
* **DB_DSN** - PostgreSQL DSN, example: "postgres://test:test@localhost/test?sslmode=disable"
* **JWT_KEY** - HS256 encryption key in BASE-64 encoding, example: "b3BlbnNza ... MEBQ=="
* **KAFKA_HOST** - comma-separated list of Kafka hosts, as used by [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go)
* **KAFKA_TOPIC** - topic name for notifications

### Runnig

With `docker-compose` it's just

```
    docker-compose up
``` 

Manual build/run:
```
    go build ./cmd/api
    ./api
```

### Authorization

API requests must be authorized with JWT tokens in "Authorization" header. Token must contain one or more of the following roles:

* **reader** - for access to Get methods
* **writer** - for access to Create, Patch, Delete methods

Token could be generated with additional tool **jwtkeygen**.

### jwtkeygen

Is an additional tool for JWT tokens generation. It uses the same JWT_KEY as API service and expects a list of roles in a command line:
```
    jwtkeygen reader writer
```

### API

API service presents methods to create, update, delete and get companies from the table.
Detailed API description see in swagger/swagger.yaml.

### Kafka notifications

Any data-modifying request results in notifications sent to Kafka topic. Notification is a JSON string with the following fields:

* **id** - UUID of changed record
* **event** - string name of event (created, updated, deleted)
* **timestamp** - UNIX-timestamp of event

