FROM golang:1.8

WORKDIR /go/src/app
COPY . .

RUN go-wrapper download  github.com/kr/pretty # "go get -d -v ./..."
RUN go-wrapper install   github.com/kr/pretty # "go install -v ./..."

CMD ["go-wrapper", "run"] # ["couch2mq"]