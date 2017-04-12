FROM debian:jessie

# This docker file is only useful if you wish to build the pi-gateway on non ARM64 platforms 
# for example (x86_64), and OSes to generate cross platform binaries that can run on the Pi 3
# using debian jessie.
#
# If you are already on a Pi 3 then all that is needed is to install the go compiler, and
# run 'go install .' inside the top level directory of the cloned github repo.
# 
# This Dockerfile fulfils one additional purpose and that is to provide an accurate record
# from development of every dependency needed to compile and run the pi-gateway
#
# To build the docker image for the build something such as the following would be used. You only
# need to do this once!. The use of the USER... variables allows the docker build to use your 
# local user id when accessing and generating the binaries.  Most OSes will have these already defined 
# for your USER the exception being Mac OSX which probably goes without saying.
#
# docker build -t magnus_build --build-arg USER=$USER --build-arg USER_ID=`id -u $USER` --build-arg USER_GROUP_ID=`id -g $USER` .
#
# To run the built container to cross compile to ARM64 after making source changes, from the top of the 
# github source repository directory
#
# docker run --cap-drop=all -it  -v `pwd`:/project -u `id -u $USER`:`id -g $GROUP` -e LOCAL_USER_ID=`id -u $USER` magnus_build
# 
MAINTAINER karlmutch@gmail.com

ENV LANG C.UTF-8

ARG USER
ENV USER ${USER}
ARG USER_ID
ENV USER_ID ${USER_ID}
ARG USER_GROUP_ID
ENV USER_GROUP_ID ${USER_GROUP_ID}

ENV INITSYSTEM on

RUN apt-get update -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y apt-utils && \
    DEBIAN_FRONTEND=noninteractive dpkg-reconfigure apt-utils && \
    apt-get install -y pulseaudio alsa-utils mplayer wget curl jq git sudo bash zsh && \
    apt-get upgrade -y && \
    apt-get dist-upgrade -y && \
    echo "exit-idle-time = -1" >> /etc/pulse/daemon.conf

# Add the build environment for compiled Go
RUN wget --quiet -O /tmp/go.tgz https://storage.googleapis.com/golang/go1.8.1.linux-amd64.tar.gz && \
    tar xzf /tmp/go.tgz

RUN echo ${USER}
RUN groupadd -f -g ${USER_GROUP_ID} ${USER}
RUN useradd -g ${USER_GROUP_ID} -u ${USER_ID} -ms /bin/bash ${USER}

USER ${USER}
WORKDIR /home/${USER}

RUN mkdir -p /home/${USER}/.ssh && \
    chmod 0700 /home/${USER}/.ssh && \
    ssh-keyscan github.com >> /home/${USER}/.ssh/known_hosts

ENV PATH=$PATH:/home/${USER}/go/bin
ENV GOROOT=/home/${USER}/go
ENV GOPATH=/project
ENV GOBIN=/project/bin

RUN cd /home/${USER} && \
    mkdir -p /home/${USER}/go && \
    tar xzf /tmp/go.tgz

VOLUME /project
WORKDIR /project

ENV GOOS linux
ENV GOARCH arm
ENV GOARM 7

CMD go get -d . && go build -o bin/pi-gateway.arm .
