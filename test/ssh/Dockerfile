FROM debian:bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        openssh-server \
        bash \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /run/sshd \
    && ssh-keygen -A

RUN echo 'root:testpassword' | chpasswd

RUN sed -i \
    -e 's/^#\?PermitRootLogin.*/PermitRootLogin yes/' \
    -e 's/^#\?PasswordAuthentication.*/PasswordAuthentication yes/' \
    /etc/ssh/sshd_config

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D", "-e"]
