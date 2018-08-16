FROM busybox

COPY assets/check /opt/resource/check
COPY assets/in /opt/resource/in
COPY assets/build-pass-fail /opt/tasks/build-pass-fail
