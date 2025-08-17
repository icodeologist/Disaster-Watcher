# Features and Roadmap
 DisasterNotifier is a microservice api written in golang that has:
 - **User authentication:** Uses jwt token based authentication
 - **Asynchronous data handling:** Fill this up
 - **Notification system:** talks to it using gRPC. It handles notificaiton events.
 - **Forward and reverse geocoding:** To handle the user location and efficiently give the co-ordinates and locaiton likewise.

 ### Roadmap
 - **Improving endpoints edge cases:**
 - **Pub-sub architecture for efficient messaging:**
 - **Separate auth microservice:**
 - **Testing:** I have a long way in writing  good tests. So ill practice and study hard for this.
 - **Improving report handles and user experience:** Currently it only displays the report to the user. In future Im planning to add reports from the external apis and validate them. Also improve the user expirience by adding more fields to report model.
## ⚡ Tech Stack

- **Language:** Go (Golang)
- **Frameword:** gin(github.com/gin-gonic/gin)
- **Database:** PostgreSQL, Redis(Lists)
- **Communication:** gRPC, Protocol Buffers
- **Auth & Security:** JWT-based authentication
- **Other Utilities:** .env


# System Design
## Flow: User Report → Notifications via gRPC
1. **User submits a report** through the API.  
2. **Report is pushed to Redis** (`Report Lists`).  
3. **Background worker (infinite loop)** continuously pops reports from Redis.  
4. For each report:  
   - Find the **affected users** using helper functions (`/services/helper.go`).  
   - Perform a **gRPC health check** on the Notification microservice.  
   - Call `SendBatchNotificationsToAffectedUsers(grpcClient, requests, report)`.  
5. **Notification Service** processes the requests and delivers alerts to users.  
![alt text](./design-pics/Disaster-redis-microservice-desing.png)


# Endpoints

## 📑 API Endpoints

### Public Endpoints
| Method | Endpoint                  | Description                        |
|--------|---------------------------|------------------------------------|
| GET    | `/report/:id`             | Get a report by its ID            |
| DELETE | `/delete/:id`             | Delete a report by its ID         |
| GET    | `/reports/all`            | Get all reports                   |
| GET    | `/reports/nearby`         | Get nearby reports (filter by location params) |

### Authentication
| Method | Endpoint                       | Description                          |
|--------|--------------------------------|--------------------------------------|
| POST   | `/auth/register`               | Register a new user                  |
| POST   | `/auth/login`                  | Login user and get JWT token         |
| GET    | `/auth/reset/password`         | Request password reset (send email)  |
| POST   | `/auth/reset/passsword`        | Handle password reset (apply change) |

### Authenticated User Endpoints *(require JWT via middleware)*
| Method | Endpoint                       | Description                           |
|--------|--------------------------------|---------------------------------------|
| POST   | `/user/api/report`             | Create a new report                   |
| GET    | `/user/profile`                | Get user profile                      |
| GET    | `/user/reports`                | Get all reports created by the user   |

# API Authentication with Curl
 Quick test with Curl.
```bash
# Register
curl -X POST http://localhost:3000/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"denzil","password":"12345678","email":"user@example.com","location":"Mumbai","phoneNumber":"9876543210"}'

# Login
curl -X POST http://localhost:3000/user/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"12345678"}'

# Profile
curl -X GET http://localhost:3000/user/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Request Password Reset
curl -X GET "http://localhost:3000/user/reset-password?email=user@example.com"

# Reset Password
curl -X POST http://localhost:3000/user/reset-password \
  -H "Content-Type: application/json" \
  -d '{"token":"reset-token","new-password":"newPass123","retype-newpassword":"newPass123"}'

```
