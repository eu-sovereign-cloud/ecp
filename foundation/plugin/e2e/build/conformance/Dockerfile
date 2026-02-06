FROM debian:trixie

WORKDIR /test

RUN apt update && \
    apt install -y wget && \
    wget https://github.com/eu-sovereign-cloud/conformance/releases/download/v1.0.0/secatest-v1.0.0 -O seca && \
    chmod +x seca 
RUN groupadd -r -g 1001 appuser && useradd -r -u 1001 -g appuser appuser
USER 1001
