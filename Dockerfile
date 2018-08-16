FROM busybox

COPY assets/check           /opt/resource/check
COPY assets/in              /opt/resource/in

COPY assets/build-pass-fail /opt/tasks/build-pass-fail

COPY assets/show-build      /opt/tasks/show-build
COPY assets/show-plan       /opt/tasks/show-plan
COPY assets/show-resources  /opt/tasks/show-resources
