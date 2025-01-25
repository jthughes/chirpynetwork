FROM debian:stable-slim

# COPY source destination
COPY chirpynetwork /bin/chirpynetwork
COPY index.html index.html

ENV PORT=8000

CMD ["/bin/chirpynetwork"]

