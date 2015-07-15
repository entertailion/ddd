FROM golang

RUN apt-get clean && apt-get update && apt-get install -y \
	bc \
	ca-certificates \
	curl \
	cython \
	g++ \
	git \
	ipython \
	libatlas-base-dev \
	libatlas-dev \
	libboost-all-dev \
	libgflags-dev \
	libgoogle-glog-dev \
	libhdf5-dev \
	libleveldb-dev \
	liblmdb-dev \
	libopencv-dev \
	libprotobuf-dev \
	libsnappy-dev \
	make \
	protobuf-compiler \
	python-dateutil \
	python-gflags \
	python-h5py \
	python-leveldb \
	python-matplotlib \
	python-networkx \
	python-nose \
	python-numpy \
	python-pandas \
	python-pil \
	python-protobuf \
	python-scipy \
	python-skimage-lib \
	python-yaml \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN curl https://bootstrap.pypa.io/get-pip.py | python

RUN pip install scikit-image

RUN git clone --depth 1 --single-branch https://github.com/BVLC/caffe.git /caffe

RUN curl http://dl.caffe.berkeleyvision.org/bvlc_googlenet.caffemodel > /caffe/models/bvlc_googlenet/bvlc_googlenet.caffemodel

RUN cd /caffe && \
	cp Makefile.config.example Makefile.config && \
	sed -i 's/# CPU_ONLY/CPU_ONLY/g' Makefile.config && \
	echo 'INCLUDE_DIRS += /usr/include/hdf5/serial' >> Makefile.config && \
	echo 'LIBRARY_DIRS += /usr/lib/x86_64-linux-gnu/hdf5/serial' >> Makefile.config && \
	make -j"$(nproc)" all pycaffe

ENV PYTHONPATH=/caffe/python
WORKDIR /ddd

COPY deepdreams.py /ddd/
COPY ddd.go /go/src/ddd/ddd.go
RUN go install ddd
RUN mkdir /images

EXPOSE 8080
ENTRYPOINT ["/go/bin/ddd"]
