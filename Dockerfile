FROM golang:latest

WORKDIR /go/src/couch2mq

RUN git clone https://github.com/mshen7310/couch2mq /go/src/couch2mq

RUN go get github.com/go-sql-driver/mysql

RUN go get github.com/kr/pretty 

RUN go get golang.org/x/crypto/ssh

RUN go get github.com/NodePrime/jsonpath

RUN go get github.com/gchaincl/dotsql

RUN cd /go/src/couch2mq

RUN go build

CMD ["./couch2mq"]