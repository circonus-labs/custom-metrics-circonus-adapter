FROM gcr.io/distroless/static
COPY adapter /
ENTRYPOINT ["/adapter"]
