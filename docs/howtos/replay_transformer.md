# Replaying already elaborated raw data

## non-idempotent transformer
If your transformer is idempotent (such as when using bdp mobility) you need to temporarily stop elaboration of new messages.  

The easiest way is to just reconfigure your transformer manually
`kubectl edit -n collector deployments/<your-transformer-name>`
Change the env variables so that the transformer listents to a new temporary queue

After changing to the new queue, delete the data you want to re-elaborate, if necessary

## Deleting time series data
When deleting time series data, make sure to also update the measurement to the most recent, as the BDP API rejects anything that's older than the the most recent measurement. An example of how to do this:
```sql
start transaction;
delete from measurementhistory h
using station s, type t
where s.id = h.station_id 
and t.id = h.type_id
and s.origin='route220' 
and t.cname = 'number-available'
and h.timestamp > to_date('20240510', 'YYYYMMDD');

update measurement m
  set double_value = t.double_value, timestamp = t.timestamp 
from (
select distinct on (h.station_id, h.type_id, h.period) h.station_id, h.type_id, h.period, h.double_value, h.timestamp
from measurementhistory h, station s, type t
where s.id = h.station_id 
and t.id = h.type_id
and s.origin='route220' 
and t.cname = 'number-available'
and h.timestamp > to_date('20240110', 'YYYYMMDD')
order by h.station_id, h.type_id, h.period, h.timestamp desc
) t
where m.station_id = t.station_id and m.type_id = t.type_id and m.period = t.period;

commit;
```

## Regenerating the notifications
Use the `/devops/encore` tool to recreate the rabbitmq messages.  
The `run.sh` file also contains the configuration

The most important things are setting the target queue, and to supply a custom mongodb query of the data you want to replay.

## Clean up
If your transformer is non-idempotent and you are using a temporary queue:
- wait until it has elaborated all the messages
- switch it back to it's original queue
- then delete the temp queue (e.g. via gui)

## Additional tooling
If you need to do some more advanced hackery with queues, the `/devops/detour` folder has some command line examples

