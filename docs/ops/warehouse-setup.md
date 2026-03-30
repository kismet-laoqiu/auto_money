# Warehouse Setup

## Bring Up Postgres / Timescale

```bash
docker compose -f deploy/docker-compose.platform.yml up -d
```

## Verify Container Readiness

```bash
docker compose -f deploy/docker-compose.platform.yml exec -T warehouse-db pg_isready -U quantlab -d quantlab
```

## Apply Warehouse Migrations

```bash
PATH=/usr/local/go/bin:$PATH ./bin/platformctl warehouse migrate \
  -config configs/platform/warehouse.yaml \
  -migrations migrations/postgres
```

## Check Warehouse Health

```bash
PATH=/usr/local/go/bin:$PATH ./bin/platformctl warehouse health \
  -config configs/platform/warehouse.yaml
```

## Roll Back / Reset

```bash
docker compose -f deploy/docker-compose.platform.yml down
rm -rf var/platform/warehouse/postgres
```
