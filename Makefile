GOFLAGS=

.PHONY: all clean

all: protocol/protocol.go build/cheetahd

# about gen protocol
protocol/protocol.go:	gen_protocol/amqp0-9-1.stripped.extended.xml gen_protocol/gen_protocol.go
	go run gen_protocol/gen_protocol.go < gen_protocol/amqp0-9-1.stripped.extended.xml | gofmt > protocol/protocol.go

# cheetahd
build/cheetahd: cheetahd/*.go
	go build${GOFLAGS} -o $@ ./cheetahd/*.go

clean:
