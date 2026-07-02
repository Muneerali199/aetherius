FROM envoyproxy/envoy:v1.30-latest
COPY envoy.yaml /etc/envoy/envoy.yaml
EXPOSE 8080 9901
CMD ["envoy", "-c", "/etc/envoy/envoy.yaml", "--log-level", "info"]
