API_URL=http://localhost:3000/api/reports
BODY='{"category":"flood","description":"test","location":"Texas, USA","priority":"high","status":"unverified"}'


LOGIN_URL=http://localhost:3000/user/login
REGI_URL=http://localhost:3000/user/register
LOGIN_BODY='{"email":"ryan@example.com", "password":"pass1234"}'
REGI_BODY='{"username":"den", "email":"den@example.com","password":"1234","location":"kalasa, India"}'

.PHONY: check-token five-con ten-sec light test-one log-den reg-den

check-token:
	@test -n "$(TOKEN)" || (echo "TOKEN is required; export it before running authenticated load tests"; exit 1)

five-con: check-token
	hey -n 5 -c 5 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

# Sustained load for 10 seconds
ten-sec: check-token
	hey -z 10s -c 50 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

# Light load sanity test
light: check-token
	hey -n 100 -c 100 \
	-m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $(TOKEN)" \
	-d $(BODY) \
	$(API_URL)

test-one: check-token
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
