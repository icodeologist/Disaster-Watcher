# Concurreny Lab -Disaster Watcher

## Experiment 1 
Worker : 2
Buffer : 4
Latency / job : 2s 
Load : hey -n 100 -c 100 instant
Not running: Saving reports to db and also not calling nominatim api(external api)

Prediction:
- 6 accepted
- 503 from 7(so 94 instant rejected)

Reality:

Well i got 500 which is unexpected

