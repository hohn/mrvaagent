all: mrvaagent

mala:
	GOOS=linux GOARCH=arm64 go build

mrvaagent: 
	go build

clean:
	rm mrvaagent

