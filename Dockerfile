FROM golang:1.14-alpine
WORKDIR /ftx-lending-bot
ADD . /ftx-lending-bot
RUN cd /ftx-lending-bot \
    && go build
ENTRYPOINT ["sh","-c","./FtxLendingBot"]
