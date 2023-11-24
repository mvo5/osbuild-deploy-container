FROM registry.fedoraproject.org/fedora:39 AS builder
RUN dnf install -y git-core golang gpgme-devel libassuan-devel
RUN dnf install -y 'dnf-command(builddep)'
RUN dnf install -y rpm-build
RUN dnf builddep -y osbuild
COPY build.sh .
RUN ./build.sh

FROM registry.fedoraproject.org/fedora:39
COPY --from=builder images/osbuild/rpmbuild/RPMS/*/*.rpm .
RUN dnf install -y *.rpm
COPY --from=builder images/osbuild-deploy-container /usr/bin/osbuild-deploy-container
COPY prepare.sh entrypoint.sh /
COPY --from=builder images/dnf-json .

ENTRYPOINT ["/entrypoint.sh"]
VOLUME /output
VOLUME /store
VOLUME /rpmmd

