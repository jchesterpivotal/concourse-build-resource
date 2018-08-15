# concourse-build-resource

Right now this is untested, so if you use it in production, shame on you.

Basically it is a gaping, bleeding security hole. Untested and dangerous. 

Kick the tires, but if the tires explode and turn your feet into abstract art, I refuse to be held responsible.

## source

* `concourse_url`: the base URL for your Concourse.
* `team`, `pipeline`, `job`: hopefully self-explanatory

Authentication is not supported currently. 

Your pipeline and jobs will need to be public or it won't work.

## in

Will produce 4 files:

* `build.json`: the build metadata
* `resources.json`: the resource versions that were involved in `get`s or `put`s
* `plan.json`: the plan of the Job
* `events.log`: the logs from the Job.

**THERE IS NO REDACTION. THIS MEANS YOUR SECRETS MAY SHOW UP IN THESE FILES. BE CAREFUL.** 

## out

No-op.

## Example

```yaml
resource_types:
- name: concourse-build
  type: docker-image
  source:
    repository: gcr.io/cf-elafros-dog/concourse-build-resource

resources:
- name: builds
  type: concourse-build
  source:
    concourse_url: https://concourse.example.com
    team: main
    pipeline: example-pipeline
    job: some-job-you-are-interested-in
    
jobs:
# ....

- name: some-job-you-are-interested-in
  public: true # required or it won't work
  plan:
  # ... whatever it is

- name: react-after-build
  public: true
  plan:
    - get: builds
      trigger: true
      version: every
    - task: echo-build
      config:
        platform: linux
        inputs:
        - name: builds
        image_resource:
          type: docker-image
          source: {repository: busybox}
        run:
          path: cat
          args: ['builds/build.json']
    - task: echo-resources
      config:
        platform: linux
        inputs:
        - name: builds
        image_resource:
          type: docker-image
          source: {repository: busybox}
        run:
          path: cat
          args: ['builds/resources.json']
    - task: echo-plan
      config:
        platform: linux
        inputs:
        - name: builds
        image_resource:
          type: docker-image
          source: {repository: busybox}
        run:
          path: cat
          args: ['builds/plan.json']
    - task: echo-log
      config:
        platform: linux
        inputs:
        - name: builds
        image_resource:
          type: docker-image
          source: {repository: busybox}
        run:
          path: cat
          args: ['builds/events.log']
```
