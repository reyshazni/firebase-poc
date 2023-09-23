init:
	go mod vendor
	echo "Project initiated"
run:
	go mod tidy
	go run ./*.go