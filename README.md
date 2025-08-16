# Disaster Watcher.
- Fully backend api written in golang. Where You can see a nearby or afar(yet to build) disasters happening and react accordingly.

## Motive
- Main motive was to learn how gRPC works
- Also try out this new idea that popped in my head. I was wondering why I rely on whatsapp group notification when people can actually post the disaster they see first or hear about and post them so that the other users who are nearby or afar notify their people. May be thats not my app. Ill think about adding this feature asap.
- I dont trust random person in my local whatsapp group and no clear precaution also need to buid this


## features
- User authorization and authentication
- User input
- User can get nearby report based on thier notification
- Basic crud operations 
- Yeah i need to add some features that should make this somewhat authentic

## Authentication

This API uses **JWT Bearer Tokens** for all requests.

### 1. Get Your Token
1. **Register** (if you’re new) → [localhost:3000/user/register](http://localhost:3000/user/register)  
2. **Log in** → [localhost:3000/user/login](http://localhost:3000/user/login)  
3. On successful login, the API will return a **JWT token** in the response.

---

### 2. Use the Token
- For every request, include the token in the `Authorization` header:
    ```
    Authorization: Bearer YOUR_JWT_TOKEN
    ```
- Example:
    ```bash
    curl -X GET http://localhost:3000/api/notifications \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6..."
    ```

---

### 3. Token Expiration
- Your JWT token is valid for **24 hours**.
- After expiration, log in again to generate a new token.

---

**Example Token**: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTQ0ODQ2NTUsImlkIjo2MH0.wo8iHgN6mSbr4lmEOiggjHw_ZfrjkyTRmucEch5XB2w


# API Authentication & Usage Guide

This API uses **JWT Bearer Tokens** for authentication.  
Add this header to all protected requests:

## Step 1 — Register
**POST** `/user/register`

```json
{
  "username": "denzil",
  "password": "mypassword",
  "email": "user@example.com",
  "location": "Mumbai",
  "phoneNumber": "9876543210"
}
```
```


## Step 2 — Login
**POST** `/user/login`
- Returns a JWT token(valid for 24hour)
```
```
{
  "username": "denzil",
  "email": "user@example.com",
  "password": "mypassword",
} 
```
```

- Response: 

{ "token": "eyJhbGciOiJIUzI1NiIs..." }

