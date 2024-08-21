FROM golang:1.22.5-alpine

WORKDIR /app

RUN apk add --no-cache git g++ openssh && \
  go install github.com/codegangsta/gin@2c98d96c9244c7426e985119b522f6e85c4bc81f
RUN mkdir /root/.ssh
RUN printf "Host gitlab.globoi.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
RUN git config --system --add safe.directory '*'
RUN git config --global url."gitlab@gitlab.globoi.com:".insteadof "https://gitlab.globoi.com"

CMD ["gin", "-b", "hydra", "run"]
