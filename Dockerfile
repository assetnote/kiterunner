FROM golang:latest

RUN apt update

RUN git clone https://github.com/assetnote/kiterunner.git

RUN cd kiterunner &&\
    make build &&\
    ln -s $(pwd)/dist/kr /usr/local/bin/kr
RUN mkdir /work
RUN mkdir /execution
ADD https://wordlists-cdn.assetnote.io/data/kiterunner/routes-large.kite.tar.gz /work/routes-large.kite.tar.gz

RUN cd /work && tar -xvzf routes-large.kite.tar.gz && rm -rf routes-large.kite.tar.gz
#RUN ls -lah /work
WORKDIR /execution
ENTRYPOINT [ "kr" ]
