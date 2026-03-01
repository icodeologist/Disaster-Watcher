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
