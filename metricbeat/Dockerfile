FROM golang:1.24.4
COPY --from=docker:26.0.0-alpine3.19 /usr/local/bin/docker  /usr/local/bin/

RUN \
    apt update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -qq -y --no-install-recommends \
         netcat-openbsd \
         python3 \
         python3-dev \
         python3-pip \
         python3-venv \
         libaio-dev \
         unzip \
         libssl-dev \
         libffi-dev \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.20.6 uses Debian 12 which now enforces PEP668.
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade PyYAML==6.0.1
RUN pip3 install requests==2.32.3
RUN pip3 install urllib3==1.26.19
RUN pip3 install docker==7.1.0
RUN pip3 install git+https://github.com/pkoutsovasilis/compose@2c4d825881a624bd87c390a13f362ae76dcd6180

# Oracle instant client
RUN cd /usr/lib \
  && curl -sLo instantclient-basic-linux.zip https://download.oracle.com/otn_software/linux/instantclient/19600/instantclient-basic-linux.x64-19.6.0.0.0dbru.zip \
  && unzip instantclient-basic-linux.zip \
  && rm instantclient-basic-linux.zip
ENV LD_LIBRARY_PATH=/usr/lib/instantclient_19_6

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
