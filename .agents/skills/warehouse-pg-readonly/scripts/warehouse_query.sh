#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: warehouse_query.sh <coverage|freshness|gaps|symbol-coverage|interval-coverage|sql> [flags]

flags:
  --provider <value>      provider filter, default comes from configs/platform/watchlist.yaml
  --symbols <csv>         comma-separated symbol filter, default comes from watchlist symbols
  --intervals <csv>       comma-separated interval filter, default comes from watchlist historical_intervals
  --all-symbols           clear the symbol filter
  --all-intervals         clear the interval filter
  --limit <n>             row limit for gap reports, default 50
  --sql "<query>"         custom read-only SQL for the sql subcommand
EOF
}

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
warehouse_config="${WAREHOUSE_CONFIG_PATH:-$repo_root/configs/platform/warehouse.yaml}"
watchlist_config="${WATCHLIST_PATH:-$repo_root/configs/platform/watchlist.yaml}"
compose_file="${WAREHOUSE_COMPOSE_FILE:-$repo_root/deploy/docker-compose.platform.yml}"

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "missing required file: $1" >&2
    exit 1
  fi
}

read_watchlist_provider() {
  awk -F': ' '/^provider: / {print $2; exit}' "$watchlist_config"
}

read_watchlist_symbols() {
  awk '
    /^symbols:/ {in_symbols=1; next}
    in_symbols && /^[^[:space:]]/ {in_symbols=0}
    in_symbols && $1 == "-" && $2 == "symbol:" {print $3}
  ' "$watchlist_config"
}

read_watchlist_intervals() {
  awk '
    /^historical_intervals:/ {in_intervals=1; next}
    in_intervals && /^[^[:space:]]/ {in_intervals=0}
    in_intervals && $1 == "-" {print $2}
  ' "$watchlist_config"
}

join_lines_csv() {
  paste -sd, -
}

validate_token() {
  local value="$1"
  if [[ ! "$value" =~ ^[A-Za-z0-9._-]+$ ]]; then
    echo "unsafe token: $value" >&2
    exit 1
  fi
}

csv_to_sql_list() {
  local csv="$1"
  local part
  local items=()
  IFS=',' read -r -a raw_parts <<< "$csv"
  for part in "${raw_parts[@]}"; do
    part=$(echo "$part" | xargs)
    [[ -n "$part" ]] || continue
    validate_token "$part"
    items+=("'$part'")
  done
  local joined=""
  local item
  for item in "${items[@]}"; do
    if [[ -n "$joined" ]]; then
      joined+=", "
    fi
    joined+="$item"
  done
  echo "$joined"
}

interval_rank_sql() {
  cat <<'EOF'
case interval
  when '1m' then 1
  when '5m' then 2
  when '15m' then 3
  when '1h' then 4
  when '4h' then 5
  when '1d' then 6
  when '1w' then 7
  else 99
end
EOF
}

build_where_clause() {
  local clauses=()
  if [[ -n "$provider" ]]; then
    validate_token "$provider"
    clauses+=("provider = '$provider'")
  fi
  if [[ -n "$symbols_csv" ]]; then
    clauses+=("symbol in ($(csv_to_sql_list "$symbols_csv"))")
  fi
  if [[ -n "$intervals_csv" ]]; then
    clauses+=("interval in ($(csv_to_sql_list "$intervals_csv"))")
  fi
  if [[ ${#clauses[@]} -eq 0 ]]; then
    echo "TRUE"
    return
  fi
  local joined=""
  local clause
  for clause in "${clauses[@]}"; do
    if [[ -n "$joined" ]]; then
      joined+=" and "
    fi
    joined+="$clause"
  done
  echo "$joined"
}

psql_exec() {
  local sql="$1"
  (
    cd "$repo_root"
    docker compose -f "$compose_file" exec -T warehouse-db \
      env PGOPTIONS='-c default_transaction_read_only=on' \
      psql -U "$db_user" -d "$db_name" -v ON_ERROR_STOP=1 -P pager=off -c "$sql"
  )
}

require_file "$warehouse_config"
require_file "$watchlist_config"
require_file "$compose_file"

dsn=$(awk -F': ' '/^dsn: / {print $2; exit}' "$warehouse_config")
if [[ -z "$dsn" ]]; then
  echo "could not read dsn from $warehouse_config" >&2
  exit 1
fi

dsn_without_scheme=${dsn#postgres://}
db_user=${dsn_without_scheme%%:*}
db_name=${dsn_without_scheme##*/}
db_name=${db_name%%\?*}
validate_token "$db_user"
validate_token "$db_name"

command="${1:-}"
if [[ -z "$command" ]]; then
  usage
  exit 2
fi
shift

provider="${WAREHOUSE_PROVIDER:-$(read_watchlist_provider)}"
symbols_csv="${WAREHOUSE_SYMBOLS:-$(read_watchlist_symbols | join_lines_csv || true)}"
intervals_csv="${WAREHOUSE_INTERVALS:-$(read_watchlist_intervals | join_lines_csv || true)}"
limit="${WAREHOUSE_LIMIT:-50}"
custom_sql=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --provider)
      provider="${2:-}"
      shift 2
      ;;
    --symbols)
      symbols_csv="${2:-}"
      shift 2
      ;;
    --intervals)
      intervals_csv="${2:-}"
      shift 2
      ;;
    --all-symbols)
      symbols_csv=""
      shift
      ;;
    --all-intervals)
      intervals_csv=""
      shift
      ;;
    --limit)
      limit="${2:-}"
      shift 2
      ;;
    --sql)
      custom_sql="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ ! "$limit" =~ ^[0-9]+$ ]]; then
  echo "limit must be numeric" >&2
  exit 1
fi

where_sql=$(build_where_clause)
interval_rank=$(interval_rank_sql)

case "$command" in
  coverage)
    psql_exec "
      select
        provider,
        symbol,
        interval,
        count(*) as row_count,
        min(open_time) as first_open_time,
        max(open_time) as last_open_time,
        round(extract(epoch from max(open_time) - min(open_time)) / 86400.0, 2) as span_days
      from market_bars
      where $where_sql
      group by provider, symbol, interval
      order by symbol asc, $interval_rank asc;
    "
    ;;
  freshness)
    psql_exec "
      select
        provider,
        symbol,
        interval,
        max(open_time) as last_open_time,
        round(extract(epoch from now() - max(open_time)) / 60.0, 2) as lag_minutes
      from market_bars
      where $where_sql
      group by provider, symbol, interval
      order by lag_minutes desc, symbol asc, $interval_rank asc;
    "
    ;;
  gaps)
    psql_exec "
      with base as (
        select
          provider,
          symbol,
          interval,
          min(open_time) as first_open_time,
          max(open_time) as last_open_time,
          count(*) as actual_rows,
          case interval
            when '15m' then 900
            when '1h' then 3600
            when '4h' then 14400
            when '1d' then 86400
            when '1w' then 604800
            else null
          end as step_seconds
        from market_bars
        where $where_sql
        group by provider, symbol, interval
      ),
      stats as (
        select
          provider,
          symbol,
          interval,
          first_open_time,
          last_open_time,
          actual_rows,
          case
            when step_seconds is null or actual_rows = 0 then null
            else floor(extract(epoch from last_open_time - first_open_time) / step_seconds)::bigint + 1
          end as expected_rows
        from base
      )
      select
        provider,
        symbol,
        interval,
        actual_rows,
        expected_rows,
        greatest(0, coalesce(expected_rows, actual_rows) - actual_rows) as gap_rows,
        first_open_time,
        last_open_time
      from stats
      order by gap_rows desc, symbol asc, $interval_rank asc
      limit $limit;
    "
    ;;
  symbol-coverage)
    psql_exec "
      with base as (
        select
          provider,
          symbol,
          interval,
          count(*) as row_count,
          min(open_time) as first_open_time,
          max(open_time) as last_open_time
        from market_bars
        where $where_sql
        group by provider, symbol, interval
      )
      select
        provider,
        symbol,
        count(*) as interval_count,
        string_agg(interval, ', ' order by case interval
          when '1m' then 1
          when '5m' then 2
          when '15m' then 3
          when '1h' then 4
          when '4h' then 5
          when '1d' then 6
          when '1w' then 7
          else 99
        end) as intervals,
        min(first_open_time) as first_open_time,
        max(last_open_time) as last_open_time,
        min(row_count) as min_rows,
        max(row_count) as max_rows
      from base
      group by provider, symbol
      order by symbol asc;
    "
    ;;
  interval-coverage)
    psql_exec "
      with base as (
        select
          provider,
          symbol,
          interval,
          count(*) as row_count,
          min(open_time) as first_open_time,
          max(open_time) as last_open_time
        from market_bars
        where $where_sql
        group by provider, symbol, interval
      )
      select
        provider,
        interval,
        count(*) as symbol_count,
        string_agg(symbol, ', ' order by symbol asc) as symbols,
        min(first_open_time) as first_open_time,
        max(last_open_time) as last_open_time,
        min(row_count) as min_rows,
        max(row_count) as max_rows
      from base
      group by provider, interval
      order by $interval_rank asc;
    "
    ;;
  sql)
    if [[ -z "$custom_sql" ]]; then
      echo "sql subcommand requires --sql" >&2
      exit 2
    fi
    psql_exec "$custom_sql"
    ;;
  *)
    echo "unknown command: $command" >&2
    usage
    exit 2
    ;;
esac
