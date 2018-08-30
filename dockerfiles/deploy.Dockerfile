FROM ubuntu:16.04
RUN apt-get update
RUN apt-get install -y ca-certificates openssl
ADD bin/appgw-ingress /
CMD ["/appgw-ingress"]