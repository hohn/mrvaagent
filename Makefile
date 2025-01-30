.PHONY: all mrvaagent mala

all: mala

mala:
	GOOS=linux GOARCH=arm64 go build

mrvaagent: 
	go build

clean:
	rm mrvaagent

