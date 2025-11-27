FROM haproxy:2.8-alpine

# Copy HAProxy configuration template
COPY haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg
COPY --chmod=755 docker-entrypoint.sh /docker-entrypoint.sh

EXPOSE 5000 5001 7000

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["haproxy", "-f", "/usr/local/etc/haproxy/haproxy.cfg"]
