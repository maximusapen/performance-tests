#!/bin/bash -e
printf "\n%s - Backup started.\n" "$(date +%T)"

backup_root_dir=/backup
influxdb_backup_dir=${backup_dir}/influxdb
grafana_backup_dir=${backup_dir}/grafana

while :; do
    case $1 in
    -i | --influxdb) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            backup_influxdb=$2
            shift
        else
            die 'ERROR: "--influxdb" requires a non-empty option argument.'
        fi
        ;;
    -g | --grafana) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            backup_grafana=$2
            shift
        else
            die 'ERROR: "--grafana" requires a non-empty option argument.'
        fi
        ;;
    -s | --source) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            source=$2
            shift
        else
            die 'ERROR: "--source" requires a non-empty option argument.'
        fi
        ;;
    -d | --destination) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            dest=$2
            shift
        else
            die 'ERROR: "--destinaton" requires a non-empty option argument.'
        fi
        ;;
    -n | --dbname) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            dbname=$2
            shift
        else
            die 'ERROR: "--dbname" requires a non-empty option argument.'
        fi
        ;;
    -t | --type) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            backup_type=$2
            shift
        else
            die 'ERROR: "--type" requires a non-empty option argument.'
        fi
        ;;
    --) # End of all options.
        shift
        break
        ;;
    -?*)
        printf 'WARN: Unknown option (ignored): %s\n' "$1" >&2
        shift
        ;;
    *) # Default case: No more options, so break out of the loop.
        break ;;
    esac

    shift
done

printf "\n - Influxdb backup: %s\n" "${backup_influx}"
printf " - Grafana backup: %s\n" "${backup_grafana}"
printf " - Source: %s\n" "${source}"
printf " - Destination: %s\n" "${dest}"
printf " - Backup type: %s\n" "${backup_type}"

if [ ${backup_grafana} == 'true' ]; then
    printf "\n - Start backup of Grafana locally to %s\n" "/backup/grafana/${backup_type}"
    mkdir -p "/backup/grafana/${backup_type}"
    rm -rf /backup/grafana/${backup_type}/* || true
    scp /var/lib/grafana/grafana.db /backup/grafana/${backup_type}
fi

if [ ${backup_influxdb} == 'true' ]; then
    printf "\n - Start backup of %s Influxdb locally to %s\n" "${dbname}" "/backup/influxdb/${backup_type}"
    mkdir -p "/backup/influxdb/${backup_type}"
    rm -rf /backup/influxdb/${backup_type}/* || true
    influxd backup -portable -database ${dbname} "/backup/influxdb/${backup_type}"
fi
printf "%s\n - Backup finished\n" "$(date +%T)"
