API_URL=http://localhost:3000/api/create
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InN1cmVzaC5uYWlrQGV4YW1wbGUuY29tIiwiZXhwIjoxNzczMTEwMDYwLCJpZCI6NywidXNlcm5hbWUiOiJzdXJlc2hfdmlqYXlhbmFnYXIifQ.5WJj0wUNmK2FuqpxZv92HQJY2dwFfYStjl8W_2Tss2Y"
BODY='{"category":"flood","description":"test","location":"Vijayanagar, Mysuru","priority":"high","status":"unverified"}'

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
