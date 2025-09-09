FROM ubuntu:latest
LABEL authors="Kovsh"

ENTRYPOINT ["top", "-b"]