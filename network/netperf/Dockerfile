# TODO Look at building from smaller base like Alpine Linux
FROM ubuntu:bionic

MAINTAINER nrockwell@us.ibm.com

# Not your dormal container. This contains a lot of tools for doing network testing
# netcat added for docker exec based ssh tunnel which relies on 'nc'
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get -y install --no-install-recommends \
    wget \
    # TODO: Do I really need curl, dnsutils
    curl \ 
    dnsutils \
    # for ping flood test
    inetutils-ping \
    iperf3 \
    # utilities
    inetutils-traceroute \
    net-tools \
    netcat \
    # for security
    supervisor &&\
    apt-get dist-upgrade -y && \
    apt-get clean &&\
    rm -Rf /var/cache/* 

# iperf3
EXPOSE 5201

# Setup supervisor
COPY programs.conf /etc/supervisor/conf.d/programs.conf

#COPY jmeter.conf /etc/supervisor/conf.d/jmeter.conf
RUN mkdir -p /root/bin/ &&\
    echo "/usr/bin/supervisord -n" > /root/bin/run_supervisor &&\
    chmod 755 /root/bin/run_supervisor

# Set password length and expiry for compliance with vulnerability advisor
RUN sed -i 's/^PASS_MAX_DAYS.*/PASS_MAX_DAYS   90/' /etc/login.defs
RUN sed -i 's/sha512/sha512 minlen=8/' /etc/pam.d/common-password

RUN useradd -ms /bin/bash perf &&\
    usermod -p '*' perf

# Run the supervisor
ENTRYPOINT ["/bin/sh", "-c"]
CMD ["/root/bin/run_supervisor -c /etc/supervisor/conf.d/programs.conf"]
