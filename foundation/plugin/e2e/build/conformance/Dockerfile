FROM debian:trixie

WORKDIR /test

RUN apt update && \
    apt install -y wget && \
    wget https://github.com/eu-sovereign-cloud/conformance/releases/download/v1.0.0/secatest-v1.0.0 -O secatest && \
    chmod +x secatest 
COPY ./foundation/plugin/e2e/build/conformance/run-conformance.sh ./
RUN chmod +x run-conformance.sh
RUN groupadd -r -g 1001 appuser && useradd -r -u 1001 -g appuser appuser
USER 1001
