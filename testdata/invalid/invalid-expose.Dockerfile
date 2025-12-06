# Invalid: EXPOSE with no port
FROM alpine:3.18
EXPOSE
CMD ["./start.sh"]
