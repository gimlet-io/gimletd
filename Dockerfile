FROM alpine:3.13

ENV DATABASE_DRIVER=sqlite3
ENV DATABASE_CONFIG=/var/lib/gimletd/gimletd.sqlite
ENV XDG_CACHE_HOME /var/lib/gimletd

ADD bin/gimletd-linux-x86_64 /bin/gimletd

RUN addgroup -S gimletd && adduser -S gimletd -G gimletd

RUN mkdir /var/lib/gimletd
RUN chown gimletd:gimletd /var/lib/gimletd

USER gimletd
WORKDIR /var/lib/gimletd

EXPOSE 8888
CMD ["/bin/gimletd"]
