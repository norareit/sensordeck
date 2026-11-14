ARG AGENTRITE_IMAGE
FROM ${AGENTRITE_IMAGE}

# The Go toolchain, from the official archive: Debian bookworm packages an older Go
# than the module needs. The checksums are those go.dev publishes for these archives.
ARG TARGETARCH
ARG GO_VERSION=1.27.4
ARG GO_SHA256_AMD64=7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840
ARG GO_SHA256_ARM64=3392582988a273a87ab5043ee1b2c9c17a2c3c00580bf9326c47a2e0aaaeac92
RUN case "${TARGETARCH}" in \
        amd64) checksum="${GO_SHA256_AMD64}" ;; \
        arm64) checksum="${GO_SHA256_ARM64}" ;; \
        *) echo "no Go archive is pinned for ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
    && archive="go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" \
    && curl -fsSL -o "/tmp/${archive}" "https://go.dev/dl/${archive}" \
    && echo "${checksum}  /tmp/${archive}" | sha256sum -c - \
    && tar -C /usr/local -xzf "/tmp/${archive}" \
    && rm "/tmp/${archive}"
ENV PATH=/usr/local/go/bin:${PATH}
# The firewall admits the module proxy but not the hosts a newer toolchain is fetched
# from, so the installed toolchain is the only one used.
ENV GOTOOLCHAIN=local
