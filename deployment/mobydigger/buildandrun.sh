#!/bin/bash
docker buildx build -f deployment/mobydigger/Dockerfile -t mobydigger:latest .
docker run -it --rm \
    --pid host \
	--cap-drop ALL \
	--cap-add CAP_SYS_ADMIN \
	--cap-add CAP_SYS_CHROOT \
	--cap-add CAP_SYS_PTRACE \
	--cap-add CAP_DAC_READ_SEARCH \
	--cap-add CAP_DAC_OVERRIDE \
	--cap-add CAP_SETUID \
	--cap-add CAP_SETGID \
	--cap-add CAP_NET_RAW \
	--security-opt systempaths=unconfined \
	--security-opt apparmor=unconfined \
	--security-opt seccomp=unconfined \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    --name mobydigger mobydigger:latest \
	--debug neighborhood-service-a-1
