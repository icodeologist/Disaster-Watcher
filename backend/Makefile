API_URL=http://localhost:3000/api/reports
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InJ5YW5AZXhhbXBsZS5jb20iLCJleHAiOjE3NzU3OTQ2NTUsImlkIjo2LCJ1c2VybmFtZSI6InJ5YW5fc2NvdHQifQ.eOes6e_AiVZJ4qSqckcgAhqKHjRQclANoQem-RJzO74"
BODY='{"category":"flood","description":"test","location":"Texas, USA","priority":"high","status":"unverified"}'


LOGIN_URL=http://localhost:3000/user/login
REGI_URL=http://localhost:3000/user/register
LOGIN_BODY='{"email":"ryan@example.com", "password":"pass1234"}'
REGI_BODY='{"username":"den", "email":"den@example.com","password":"1234","location":"kalasa, India"}'

five-con:
	hey -n 5 -c 5 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

# Sustained load for 10 seconds
ten-sec:
	hey -z 10s -c 50 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

# Light load sanity test
light:
	hey -n 20 -c 5 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

test-one:
	hey -n 1 -c 1 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

log-den:
	curl POST $(LOGIN_URL) \
		-H "Content-Type: application/json" \
		-d $(LOGIN_BODY) \
		-v
reg-den:
	curl -X POST $(REGI_URL) \
		-H "Content-Type: application/json" \
		-d $(REGI_BODY) \
		-v 
