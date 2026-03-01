API_URL=http://localhost:3000/api/create
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImRlbjFAZXhhbXBsZS5jb20iLCJleHAiOjE3NzIzNDYwNTksImlkIjozLCJ1c2VybmFtZSI6ImRlbjEifQ.ne1FkF-ySexMafd0jn9Rnj90rUX33CqZn3XWPPw0NtY"
BODY='{"category":"flood","description":"test","location":"test","priority":"high","status":"unverified"}'

one-time:
	hey -n 100 -c 100 \
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
