FROM debian:trixie

WORKDIR /test

RUN apt update && \
    apt install -y curl jq wget && \
    wget https://github.com/eu-sovereign-cloud/conformance/releases/download/v1.0.1.beta8/secatest-v1.0.1.beta8-amd64 && \
    chmod +x secatest* && \
    ln -s ./secatest-v1.0.1.beta8-amd64 secatest
COPY ./foundation/plugin/e2e/build/conformance/run-conformance.sh ./
RUN chmod +x run-conformance.sh
RUN groupadd -r -g 1001 appuser && useradd -r -u 1001 -g appuser appuser
USER 1001
