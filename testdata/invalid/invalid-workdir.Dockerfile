# Invalid: WORKDIR with no path
FROM alpine:3.18
WORKDIR
CMD ["./start.sh"]
