FROM debian:trixie AS packages

# create neccassry folder structure
RUN mkdir -p /export/usr/bin

# update apt and install dependencies
RUN apt-get update
RUN apt-get install -y webp ffmpeg

# copy WebP binaries
RUN cp /usr/bin/cwebp /export/usr/bin/
RUN cp /usr/bin/dwebp /export/usr/bin/
RUN cp /usr/bin/img2webp /export/usr/bin/
RUN cp /usr/bin/gif2webp /export/usr/bin/
RUN cp /usr/bin/vwebp /export/usr/bin/
RUN cp /usr/bin/webpinfo /export/usr/bin/
RUN cp /usr/bin/webpmux /export/usr/bin/

# copy ffmpeg binaries
RUN cp /usr/bin/ffmpeg /export/usr/bin/
RUN cp /usr/bin/ffprobe /export/usr/bin/

# copy libraries
RUN ldd /usr/bin/cwebp | awk '{print $3}' | grep '^/' | xargs -I '{}' cp --parents -v '{}' /export/
RUN cp --parents "$(ldd /usr/bin/cwebp | grep 'ld-linux' | awk '{print $1}')" /export/

RUN ldd /usr/bin/ffmpeg | awk '{print $3}' | grep '^/' | xargs -I '{}' cp --parents -v '{}' /export/
RUN cp --parents "$(ldd /usr/bin/ffmpeg | grep 'ld-linux' | awk '{print $1}')" /export/

RUN ldd /usr/bin/ffprobe | awk '{print $3}' | grep '^/' | xargs -I '{}' cp --parents -v '{}' /export/
RUN cp --parents "$(ldd /usr/bin/ffprobe | grep 'ld-linux' | awk '{print $1}')" /export/

FROM scratch

ARG TARGETPLATFORM

ENTRYPOINT ["/usr/bin/amur"]

COPY $TARGETPLATFORM/amur /usr/bin/

EXPOSE 3000