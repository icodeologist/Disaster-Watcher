API_URL=http://localhost:3000/api/reports
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImRlbkBleGFtcGxlLmNvbSIsImV4cCI6MTc3Mzk2ODYzMiwiaWQiOjIsInVzZXJuYW1lIjoiZGVuIn0.WUB81fAuGHgxS-WIIRHm6BQawKPKK1Nkm3rXvIceQj4"
BODY='{"category":"flood","description":"test","location":"Vijayanagar, Mysuru","priority":"high","status":"unverified"}'


LOGIN_URL=http://localhost:3000/user/login
LOGIN_BODY='{"email":"den@example.com", "password":"pichu"}'

one-time:
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
