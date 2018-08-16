FROM busybox

COPY assets/check /opt/resource/check
COPY assets/in /opt/resource/in
COPY assets/build-pass-fail /opt/tasks/build-pass-fail
COPY assets/show-plan /opt/tasks/show-plan
