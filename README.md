# concourse-build-resource

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

I can't prevent you from leaking secrets this way. But if your builds are already public and leaking secrets,
that is sad and distressful.

## out

No-op.

# Utility tasks

Some convenience tasks are included to help you make quick and easy use of the resource.

The default input to the tasks is `build`, but you can use 
[input mapping](https://concourse-ci.org/task-step.html#input_mapping) to rename this input.

## `build-pass-fail`

This task consumes the `build` folder output from the resource and itself passes or fails depending 
on the results of the build being watched.

This is useful if you coordinate with downstream teams who consume your work: you can add a job to your pipeline
which fails when the upstream fails.

## `show-build`, `show-plan`, `show-resources`

These tasks produce pretty-printed output of the build, plan and resource JSON files.

## `show-logs`

Produces the logs from the build, including colouring (which is retained in Concourse's logs).

To avoid confusion, the log being printed is wrapped with "begin log" and "end log" lines.

## Example

```yaml
resource_types:
- name: concourse-build
  type: docker-image
  source:
    repository: gcr.io/cf-elafros-dog/concourse-build-resource

resources:
- name: build
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
    - get: build
      trigger: true
      version: every
    - task: pass-if-the-build-passed
      file: concourse-build-resource/tasks/build-pass-fail/task.yml
    - task: show-build
      file: concourse-build-resource/tasks/show-build/task.yml
    - task: show-plan
      file: concourse-build-resource/tasks/show-plan/task.yml
    - task: show-resources
      file: concourse-build-resource/tasks/show-resources/task.yml
    - task: show-logs
      file: concourse-build-resource/tasks/show-logs/task.yml
```

## Contributing

I'll be _very_ grateful if you write some tests first, given how much of a hassle it was to backfill them.

I'm using [spec](https://github.com/sclevine/spec) to organise the tests and
[Gomega](https://github.com/onsi/gomega) for matchers and utilities.
