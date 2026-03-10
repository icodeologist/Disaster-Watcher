User submits report
       ↓
Gin saves to DB → returns "Report received" to user immediately
       ↓
Spawns gRPC call async (user doesn't wait for this)
       ↓
gRPC orchestrates 3 tasks concurrently via worker pool:
       ↓
  [Worker 1]          [Worker 2]          [Worker 3]
  Verify user         Check report        Calculate
  score/credibility   for fake signals    affected radius
       ↓                   ↓                   ↓
         All results collected (wait for all 3)
                      ↓
              Calculate report score
                      ↓
         If score good → find affected users
                      ↓
              Worker pool sends emails
              (100 users = 100 concurrent workers)




Report pipeline
- User posts report
- Save to DB
- Push the whole report struct to reportChannel - 1st QUEUE 
- 2 worker pools 
- Extract affected user IDS - 1st worker pool
- Push the affected users to AffectedusersIds Channel - 2nd QUEUE
- Send notification to affectedUsers - 2 worker pool
Main wires. Create channel and server
Workers work. Do the actual function
Handlers enqueue. Add all neccessary things to queue


