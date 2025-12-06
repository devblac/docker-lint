# Invalid: Missing FROM instruction
RUN apt-get update
COPY . /app
CMD ["./start.sh"]
