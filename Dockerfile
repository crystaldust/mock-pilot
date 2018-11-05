FROM alpine
WORKDIR /root/
COPY ./mock-pilot ./mock-pilot
COPY ./tests ./tests
RUN chmod +x ./mock-pilot
CMD ["./mock-pilot"]
