FROM debian:wheezy

MAINTAINER The Protogalaxy Project

EXPOSE 9090

ENTRYPOINT ["./main", "-logtostderr", "-v=4"]

COPY ./target/bin/main .
