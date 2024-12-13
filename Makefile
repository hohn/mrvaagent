all: mrvaagent

mrvaagent: 
	# GOOS=linux GOARCH=arm64 go build
	go build

clean:
	rm mrvaagent

