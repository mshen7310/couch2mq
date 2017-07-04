FROM golang:latest

WORKDIR /go/bin

RUN git clone https://github.com/mshen7310/couch2mq /go/src/couch2mq

RUN go get github.com/kr/pretty 

RUN go get golang.org/x/crypto/ssh

RUN go get github.com/NodePrime/jsonpath

RUN go get github.com/gchaincl/dotsql

RUN cd /go/src/couch2mq

RUN go install

RUN cp conf.json /go/bin

CMD ["couch2mq"]