cov:
	go test -coverprofile=cover.out ./... 
	go tool cover -html=cover.out

e2e:
	go test ./adb -run TestE2E -v