#!/usr/bin/env bash
docker run -v "$(pwd):/work" uber/prototool:latest prototool generate
