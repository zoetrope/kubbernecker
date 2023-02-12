FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.source https://github.com/zoetrope/kubbernecker
COPY kubbernecker-metrics /
USER 65532:65532

ENTRYPOINT ["/kubbernecker-metrics"]
