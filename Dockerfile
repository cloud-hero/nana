FROM  golang:latest

WORKDIR $GOPATH/src/github.com/cloud-hero/nana

COPY Gopkg.* ./

RUN go get -u github.com/golang/dep/cmd/dep

RUN dep ensure -vendor-only

ADD . .
