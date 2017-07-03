FROM scratch
COPY bin/main /main
ENTRYPOINT ["/main"]