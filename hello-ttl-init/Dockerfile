FROM alpine:3.8
RUN apk add --no-cache curl
RUN adduser -D init
USER init
CMD ["./register-consul-service.sh"]