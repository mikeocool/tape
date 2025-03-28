FROM node:20-alpine

RUN npm install -g @devcontainers/cli

RUN apk add --update docker openrc
RUN rc-update add docker boot

RUN mkdir /workspace
WORKDIR /workspace


# ENTRYPOINT ["devcontainer-wrapper"]
