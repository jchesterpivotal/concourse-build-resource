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

# Utility tasks

Some convenience tasks are included to help you make quick and easy use of the resource.

The default input to the tasks is `build`, but you can use 
[input mapping](https://concourse-ci.org/task-step.html#input_mapping) to rename this input. See the example.

## `build-pass-fail`

This task consumes the `build` folder output from the resource and itself passes or fails depending 
on the results of the build being watched.

This is useful if you coordinate with downstream teams who consume your work: you can add a job to your pipeline
which fails when the upstream fails.

# `show-plan`

Outputs a pretty-printed version of the `plan.json`.

## Example

```yaml
resource_types:
- name: concourse-build
  type: docker-image
  source:
    repository: gcr.io/cf-elafros-dog/concourse-build-resource

resources:
- name: build-from-elsewhere
  type: concourse-build
  source:
    concourse_url: https://concourse.example.com
    team: main
    pipeline: example-pipeline
    job: some-job-you-are-interested-in

- name: concourse-build-resource
  type: git
  source: {uri: https://github.com/jchesterpivotal/concourse-build-resource.git}

jobs:
# ....

- name: some-job-you-are-interested-in
  public: true # required or the resource won't work
  plan:
  # ... whatever it is

- name: react-after-build
  public: true
  plan:
    - get: concourse-build-resource # for task YAML
    - get: build-from-elsewhere
      trigger: true
      version: every
    - task: pass-if-the-build-from-elsewhere-passed
      file: concourse-build-resource/tasks/build-pass-fail/task.yml
      input_mapping: {build: build-from-elsewhere} 
    - task: show-plan
      file: concourse-build-resource/tasks/show-plan/task.yml
      input_mapping: {build: build-from-elsewhere}
```
