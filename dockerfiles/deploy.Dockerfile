FROM ubuntu:16.04
RUN apt-get update
RUN apt-get install -y ca-certificates openssl
RUN useradd appgw-ingress-user
ADD bin/appgw-ingress /
RUN chown appgw-ingress-user /appgw-ingress
USER appgw-ingress-user
RUN chmod +x /appgw-ingress
CMD ["/appgw-ingress"]
